CC      	= go
PROGRAM		= migrator
BUILDDIR	= build
prefix		= /usr

.PHONY: build clean cleanBuild distclean run install uninstall

build: $(PROGRAM)

$(PROGRAM):
	$(CC) build

clean:
	rm -f $(PROGRAM)

cleanBuild:
	rm -rf $(BUILDDIR)

distclean: clean

run: cleanBuild clean build

# https://www.gnu.org/software/make/manual/html_node/DESTDIR.html
install:
	install -D -m 0755 $(PROGRAM) $(DESTDIR)$(prefix)/bin/$(PROGRAM)

uninstall:
	-rm -f $(DESTDIR)$(prefix)/bin/$(PROGRAM)

