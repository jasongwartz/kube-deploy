FROM golang:alpine AS builder

WORKDIR /go/src/github.com/mycujoo/kube-deploy
RUN apk update && apk add git
COPY . .
RUN  go get -d -v .
#RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .
RUN go build .
COPY /go/src/github.com/mycujoo/kube-deploy/kube-deploy ./asdfasdf

# FROM alpine
# RUN apk update && apk add git
# RUN addgroup -S kube-deploy && adduser -S -g kube-deploy kube-deploy

# USER kube-deploy
# WORKDIR /src

# COPY deploy.yaml deploy.yaml
# COPY --from=0 /go/src/github.com/mycujoo/kube-deploy/kube-deploy /kube-deploy
# CMD ["/kube-deploy"]
