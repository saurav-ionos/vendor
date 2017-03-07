
all: docker

svr:
	go build 

deploy: svr
	echo "Deploying .... "
	mkdir -p docker/apisvr
	cp apisvr docker/apisvr
	cp apisvr.sh   docker/apisvr

docker: deploy
	echo "Creating docker image"
	make -C docker

clean:
	echo "Cleaning "
	go clean 
	rm -rf docker/apisvr
