from os import listdir,remove
from os.path import isfile,join
from time import sleep
from subprocess import Popen,PIPE
import threading

#The destinations we need to send to
dest_map = {}
dest_map["ded731565cef841830b3160d068cbb55"] = "172.20.116.2"
dest_map["d41d8cd98f00b204e9800998ecf8427e"] = "172.20.104.2"

lookup_path="/mnt/ftp/1/"

def do_scp(host, srcpath, destpath):
    #start scp of the file for the file to the destination
    copy_path = destpath+"hidden"
    cmd = "scp " + srcpath + " ionos@" + host + ":" + copy_path
    op = Popen(cmd, shell=True, stdout=PIPE)
    op.wait()
    rename_cmd = "ssh ionos@" + host + " mv " + copy_path +  " " + destpath
    print rename_cmd
    op = Popen(rename_cmd, shell=True, stdout=PIPE)
    op.wait()
    
    print "file ", srcpath, "sent to ", host 

    
while True:
    onlyfiles = [join(lookup_path, f) for f in listdir(lookup_path) \
                 if isfile(join(lookup_path, f)) and not f.startswith(".")]
    
    for x in onlyfiles:
        #ship them to the destinations
        cmd = "~/chunkdecode --ship "+ x
        op = Popen(cmd, shell=True, stdout=PIPE)
        tlist = []
        for dest in op.stdout.readlines():
            key = dest.rstrip()
            if key:
                tlist.append(threading.Thread(target=do_scp, args=(dest_map[key], x, x)))
        op.wait()
                
        for t in tlist:
            t.start()
                    
        for t in tlist:
            t.join()

        remove(x)
                        
        sleep(1)
