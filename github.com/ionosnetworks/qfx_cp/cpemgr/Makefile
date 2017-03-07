
all: docker

svr:
	go build 

deploy: svr
	echo "Deploying .... "
	mkdir -p docker/cpemgr
	cp cpemgr docker/cpemgr
	cp cpemgr.sh   docker/cpemgr

docker: deploy
	echo "Creating docker image"
	make -C docker

clean:
	echo "Cleaning "
	go clean 
	rm -rf docker/cpemgr
