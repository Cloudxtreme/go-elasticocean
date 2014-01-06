default: clean build

build:
	go get -v -d -u
	go build -v -o eo

pkg: build
	mkdir $(PACKAGE)
	mv eo $(PACKAGE)
	tar -cvzf $(TGZ) $(PACKAGE)
	rm -rf $(PACKAGE)

install:
	cp eo $(GOBIN)

d:
	go build -v -o eo

.PHONY: clean
clean:
	-rm eo
