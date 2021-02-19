# hello

## how to build
```sh
docker build -f config/docker/hellosvc/Dockerfile -t yornesek/hellosvc:$(git rev-parse --short HEAD) -t yornesek/hellosvc:latest .
docker push yornesek/hellosvc -a
```
## how to deploy
```sh
kubectl -n hello rollout restart deploy hello
```