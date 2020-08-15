IMAGE=docker.jw4.us/vanity

all: build

build:
	docker build -t $(IMAGE) .

run: build
	docker run -p 39999:39999 --rm $(IMAGE)

push: build
	docker push $(IMAGE)
