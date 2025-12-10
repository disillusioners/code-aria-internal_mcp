.PHONY: build-mcp-servers install-mcp-servers clean-mcp-servers mcp-servers

# Detect operating system and include the appropriate Makefile
ifeq ($(OS),Windows_NT)
    include Makefile.windows
else
    include Makefile.unix
endif
