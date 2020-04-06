.PHONY: all clean
vcregister_bin = vcregister-linux vcregister-osx
vcbench_bin = vcbench-linux vcbench-osx

all: vcregister vcbench

vcregister: 
	$(MAKE) -C cmd/vcregister
	$(foreach bin,$(vcregister_bin),mv cmd/vcregister/$(bin) bin;)
	
vcbench:
	$(MAKE) -C cmd/vcbench
	$(foreach bin,$(vcbench_bin),mv cmd/vcbench/$(bin) bin;)

clean:
	-$(foreach bin, $(vcregister_bin),rm bin/$(bin);)
	-$(foreach bin, $(vcbench_bin),rm bin/$(bin);)
