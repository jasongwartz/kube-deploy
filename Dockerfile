FROM golang:alpine

WORKDIR /go/src/github.com/mycujoo/kube-deploy
COPY . .
RUN apk update && apk add git
RUN  go get -d -v .
#RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .
RUN go build .

FROM alpine
RUN addgroup -S kube-deploy && adduser -S -g kube-deploy kube-deploy

USER kube-deploy
WORKDIR /src

COPY --from=0 /go/src/github.com/mycujoo/kube-deploy/kube-deploy /kube-deploy
CMD ["/kube-deploy"]
