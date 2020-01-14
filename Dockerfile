FROM golang
COPY ./ /build/
WORKDIR /build
RUN go build
CMD ./dynamic-attach-go