# Reddit

### Запуск БД

```
docker-compose up
```

### Запуск тестов
```
go test -v -coverpkg ./... ./... -coverprofile=cover.out.tmp && cat cover.out.tmp | grep -e "mongo_repo.go" -e "mode" -e "mysql_repo.go" -e "authorization.go" -e "post.go" > cover.out && go tool cover -html=cover.out -o cover.html
```