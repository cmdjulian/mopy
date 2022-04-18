```bash
docker build --ssh -t my-python-app:latest -f PyDockerfile.yaml .
docker run --rm my-python-app:latest
```