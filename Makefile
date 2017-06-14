
generate: build_js
	go generate

build_js:
	coffee -c application.coffee
