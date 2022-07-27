include *.make

.PHONY: integration
integration:
	$(MAKE) -C ./integration test