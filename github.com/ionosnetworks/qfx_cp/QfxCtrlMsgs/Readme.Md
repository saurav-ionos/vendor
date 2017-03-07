# using flatbuf

Follow these steps to get flatbuf compiled.

sudo apt-get install cmake

go get github.com/google/flatbuffers  
cd github.com/google/flatbuffers  
cmake CMakeLists.txt  
make   

This will build flatbuf compiler "flatc" which is used to compile and generate stubs for various languages. We will be using "go".

Executing "make" in this folder will generate stubs.
