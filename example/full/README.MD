```bash
docker build --build-arg BUILDKIT_INLINE_CACHE=1 --rm --ssh -t my-python-app:latest -f PyDockerfile.yaml .
```