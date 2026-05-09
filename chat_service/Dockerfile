FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY main.py .

ENV PORT=8080
ENV WISDOM_ENGINE_URL=http://wisdom-engine:8080

CMD ["python", "main.py"]
