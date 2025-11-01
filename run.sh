go build -o webserver ./cmd/webserver \
&& ./webserver -addr :8080 -store data
