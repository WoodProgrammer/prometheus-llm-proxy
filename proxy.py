from flask import Flask, Response, jsonify
import gzip
import json
from io import BytesIO

app = Flask(__name__)

@app.route('/api/v1/query_range', methods = ['GET', 'POST'])
def compressed_json():
    # JSON verisi
    data = {"status":"success","data":{"resultType":"vector","result":[{"metric":{"__name__":"up","instance":"localhost:9090","job":"prometheus"},"value":[1753744320.625,"1"]}]}}

    # JSON'u string'e çevir
    json_data = json.dumps(data)

    # Gzip ile sıkıştır
    buffer = BytesIO()
    with gzip.GzipFile(fileobj=buffer, mode='w') as gz_file:
        gz_file.write(json_data.encode('utf-8'))

    # Sıkıştırılmış içeriği al
    gzipped_content = buffer.getvalue()

    # Flask Response nesnesi ile dön
    response = Response(gzipped_content)
    response.headers['Content-Type'] = 'application/json'
    response.headers['Content-Encoding'] = 'gzip'
    response.headers['Vary'] = 'Origin'
    response.headers.pop('Connection', None)  

    return response

if __name__ == '__main__':
    app.run(debug=True, port=8000)