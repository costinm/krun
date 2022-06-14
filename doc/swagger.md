# Swagger 

You can define an API with proto - and generate an OpenAPI, or use any other 'first class' definition.

From OpenAPI there are multiple generators - including Swagger. It can generate bash, go, rust, etc.

- bash is based on curl
- heavy API (go-openapi) , some old


Using it:

kubectl proxy --port=8080 &
curl localhost:8080/openapi/v2 > k8s-swagger.json
docker run     --rm     -d     -p 80:8080     -e SWAGGER_JSON=/k8s-swagger.json     -v $(pwd)/k8s-swagger.json:/k8s-swagger.json     swaggerapi/swagger-ui
docker run -rm -u $(id -u) -v $(pwd):/local   -v $(pwd)/k8s-swagger.json:/k8s-swagger.json  swaggerapi/swagger-codegen-cli generate -l go -o /local/go -i /k8s-swagger.json



alias swagger='docker run --rm -it  --user $(id -u):$(id -g) -e GOPATH=$(go env GOPATH):/go -v $HOME:$HOME -w $(pwd) quay.io/goswagger/swagger'
