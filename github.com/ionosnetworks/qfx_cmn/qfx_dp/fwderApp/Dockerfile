FROM ubuntu:14.04 
EXPOSE 22

ADD fwderApp /home/ionos/fwderApp
ADD nh.txt /var/ionos/nh.txt
RUN mkdir -p /mnt/ftp/1
#ENTRYPOINT /home/ionos/fwderApp
CMD ["/home/ionos/fwderApp"]
