DOCKER_IMAGE = "googlegemini/proxy-to-gemini"

build:
	docker build -t $(DOCKER_IMAGE) .

publish: build
	docker push $(DOCKER_IMAGE)

run: build
	docker run -p 5555:5555 -e GEMINI_API_KEY=${GEMINI_API_KEY} $(DOCKER_IMAGE)