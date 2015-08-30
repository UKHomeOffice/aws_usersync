FROM golang:1.4


# Configure Go
ENV GOPATH /go
ENV PATH /go/bin:$PATH

# Install gb
RUN mkdir -p ${GOPATH}/{src,bin} ;\
    go get github.com/constabulary/gb/... ;\
    mv /go/bin/gb /bin

WORKDIR /go

CMD ["gb build all"]
