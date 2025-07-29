import re
from fastapi import FastAPI, Request, Response
from fastapi.responses import JSONResponse, StreamingResponse
import httpx
import uvicorn
from starlette.types import ASGIApp, Receive, Scope, Send
from fastapi.middleware.gzip import GZipMiddleware
import json
from io import BytesIO

class CapitalizeHeadersMiddleware:

    def __init__(self, app: ASGIApp):
        self.app = app

    async def __call__(self, scope: Scope, receive: Receive, send: Send):
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return

        async def send_wrapper(message):
            if message["type"] == "http.response.start":
                headers = []
                for name, value in message["headers"]:
                    # Capitalize headers:
                    #   'x-cat-dog' -> 'x-Cat-Dog'
                    decoded_name = name.decode("latin1")
                    capitalized = "-".join(
                        part.capitalize() for part in decoded_name.split("-")
                    )
                    headers.append((capitalized.encode("latin1"), value))
                message["headers"] = headers
            await send(message)

        await self.app(scope, receive, send_wrapper)



app = FastAPI()
app.add_middleware(CapitalizeHeadersMiddleware)
#app.add_middleware(GZipMiddleware)
PROMETHEUS_URL = "http://localhost:9090"


def extract_promql(text: str) -> str:
    # ``` ile başlamış kod varsa onu al
    match = re.search(r"```(?:promql)?\n(.+?)```", text, re.DOTALL)
    if match:
        return match.group(1).strip()
    
    # Yoksa tek satırlık query varsa onu al
    match = re.search(r"(rate\(.+?\)|avg_over_time\(.+?\)|sum\(.+?\)|count\(.+?\))", text)
    if match:
        return match.group(1).strip()

    return text.strip()


@app.get("/api/v1/label/__name__/values")
async def passthrough_label_values(request: Request):
    params = dict(request.query_params)
    url = f"{PROMETHEUS_URL}/api/v1/label/__name__/values"

    async with httpx.AsyncClient() as client:
        prom_response = await client.get(url, params=params)

    return JSONResponse(
        status_code=prom_response.status_code,
        content=prom_response.json()
    )

@app.get("/api/v1/labels")
async def passthrough_label_values(request: Request):
    params = dict(request.query_params)
    url = f"{PROMETHEUS_URL}/api/v1/labels"

    async with httpx.AsyncClient() as client:
        prom_response = await client.get(url, params=params)

    return JSONResponse(
        status_code=prom_response.status_code,
        content=prom_response.json()
    )

@app.get("/api/v1/label/que/values")
async def que_values(request: Request):
    params = dict(request.query_params)
    url = f"{PROMETHEUS_URL}/api/v1/label/que/values"

    async with httpx.AsyncClient() as client:
        prom_response = await client.get(url, params=params)

    return JSONResponse(
        status_code=prom_response.status_code,
        content=prom_response.json()
    )


@app.api_route("/api/v1/query_range", methods=["GET", "POST"])
async def intercept_query(request: Request):
    async with httpx.AsyncClient() as client:
        # Prometheus’a gerçek query’yi gönder
        #headers = {"Accept-Encoding": "gzip"}  
        prom_response = await client.get(
            "http://localhost:9090/api/v1/query?query=up"
            #headers=headers,
            #follow_redirects=True,
        )

        print("The header is ", prom_response.headers)
        headers = dict(prom_response.headers)
        headers["Content-Length"] = str(len(prom_response.content))
        headers["Content-Type"]="text/plain"
        EXCLUDED_HEADERS = {"content-length", "transfer-encoding", "connection"}

        safe_headers = {
            k: v
            for k, v in prom_response.headers.items()
            if k.lower() not in EXCLUDED_HEADERS
        }


        return Response(
            content=prom_response.read(),
            status_code=prom_response.status_code,
            headers=safe_headers,
        )

    gzip.compress(data)
    
    #JSONResponse(content=prom_response.json())
if __name__ == "__main__":
    uvicorn.run("main:app", host="0.0.0.0", port=8000, reload=True, proxy_headers=True)