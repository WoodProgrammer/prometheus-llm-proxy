import re
from fastapi import FastAPI, Request, Response
import httpx
import uvicorn

app = FastAPI()
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


@app.api_route("/api/v1/labels", methods=["GET", "POST"])
async def catch_all_prometheus_api(request: Request, path: str):
    async with httpx.AsyncClient() as client:
        # Orijinal istek bilgileri
        method = request.method
        headers = dict(request.headers)
        body = await request.body()
        params = dict(request.query_params)

        # Prometheus'a yönlendir
        upstream = f"{PROMETHEUS_URL}/api/v1/{path}"
        resp = await client.request(
            method=method,
            url=upstream,
            headers=headers,
            params=params,
            content=body
        )

        # JSON döndür
        try:
            return resp.json()
        except Exception:
            return Response(content=resp.content, status_code=resp.status_code, media_type=resp.headers.get("content-type"))
        

@app.api_route("/api/v1/query_range", methods=["GET", "POST"])
async def intercept_query(request: Request):
    query = ""
    if request.method == "GET":
        query = request.query_params.get("query", "")
    elif request.method == "POST":
        content_type = request.headers.get("content-type", "")
        if "application/x-www-form-urlencoded" in content_type:
            body = await request.body()
            from urllib.parse import parse_qs
            parsed = parse_qs(body.decode())
            query = parsed.get("query", [""])[0]
    
    print("The query is ", query)
    params = dict(request.query_params)

    if "llm_dashboard_metric" in query:
        match = re.search(r'llm_dashboard_metric\{query="([^"]+)"\}', query)
        print("The match result is ", match)
        if match:
            natural_query = match.group(1)
            print("🔥 Intercepted natural query:", natural_query)

            # LLM'e gönder
            async with httpx.AsyncClient() as client:

                prom_response = await client.get(
                    f"http://localhost:9090/api/v1/query?query=up"
                )
                return prom_response.json()

if __name__ == "__main__":
    uvicorn.run("main:app", host="0.0.0.0", port=8000, reload=True)