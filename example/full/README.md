To clone the ssh dependency, you have to have a valid ssh key already added into your ssh agent.

```bash
docker build --ssh -t my-python-app:latest -f Mopyfile.yaml .
docker run --rm my-python-app:latest
```

[Dockerfile](Dockerfile) contains a beautified rendering of the `Mopyfile`. 