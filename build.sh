go get -d -v .

for GOOS in darwin linux; do
    echo "\n\n=> Building for $GOOS\n"
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=amd64 go build -a -v -installsuffix cgo .
    echo "\n\n=> Pushing to S3 for $GOOS\n"
    aws s3 cp kube-deploy s3://binary-distribution/kube-deploy-$GOOS-amd64 --acl public-read
done