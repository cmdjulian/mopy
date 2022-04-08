```bash
docker build --build-arg BUILDKIT_INLINE_CACHE=1 --rm -t env:latest -f PyDockerfile.yaml .
```