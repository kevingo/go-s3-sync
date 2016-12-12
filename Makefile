BINARY=sync
GOOS=linux
GOARCH=amd64

test:
	go test

local:
	go build -o ${BINARY} *.go

clean:
	if [ -a ${BINARY} ] ; \
	then \
		rm ${BINARY} ; \
	fi;

prod:
	GOOS=${GOOS} GOARCH=${GOARCH} go build -o sync *.go