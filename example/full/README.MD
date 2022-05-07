```bash
docker build --ssh -t my-python-app:latest -f Mopyfile.yaml .
docker run --rm my-python-app:latest
```