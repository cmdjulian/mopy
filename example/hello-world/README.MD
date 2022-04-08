```bash
docker build --build-arg BUILDKIT_INLINE_CACHE=1 --rm -t hello-world:latest -f PyDockerfile.yaml .
```