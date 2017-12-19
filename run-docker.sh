docker run -it \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v ~/.kube:/home/kube-deploy/.kube \
    -v $PWD:/src\
    mycujoo/kube-deploy $@
