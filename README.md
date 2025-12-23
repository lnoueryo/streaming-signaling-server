```
docker build -t streaming-signaling -f docker/Dockerfile.dev .
```
```
docker run \
  --name streaming-signaling-dev \
  -p 8080:8080 \
  -v $(pwd):/app \
  -v /app/tmp \
  streaming-signaling \
  air -c .air.toml
```