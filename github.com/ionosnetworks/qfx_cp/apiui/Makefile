
all: docker

svr:
	go build 

deploy: svr
	echo "Deploying .... "
	mkdir -p docker/apiui 
	cp apiui  docker/apiui 
	cp apiui.sh   docker/apiui 

docker: deploy
	echo "Creating docker image"
	make -C docker

clean:
	echo "Cleaning "
	go clean 
	rm -rf docker/apiui 
