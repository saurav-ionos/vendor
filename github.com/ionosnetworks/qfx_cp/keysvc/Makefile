
all: docker

svr:
	go build 

deploy: svr
	echo "Deploying .... "
	mkdir -p docker/keysvc
	cp keysvc docker/keysvc
	cp keysvc.sh   docker/keysvc

docker: deploy
	echo "Creating docker image"
	make -C docker

test:
	make -C msgclitest

clean:
	echo "Cleaning "
	go clean 
	rm -rf docker/keysvc
