FROM python:3.9-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Generate Python code from protobuf definitions
RUN python -m grpc_tools.protoc -I./protos --python_out=. --grpc_python_out=. ./protos/ai_extension.proto

EXPOSE 50051

CMD ["python", "-m", "src.main"]