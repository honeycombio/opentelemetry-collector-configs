FROM cimg/go:1.17

ENV PYENV_ROOT=/home/circleci/.pyenv \
	PATH=/home/circleci/.pyenv/shims:/home/circleci/.pyenv/bin:/home/circleci/.poetry/bin:$PATH \
	OTC_BUILDER_VERSION=0.50.0

RUN sudo apt-get update -qq && \
	sudo apt-get install -y \
	python \
	libbz2-dev \
	libreadline-dev \
	libssl-dev \
	python-openssl && \
	curl https://pyenv.run | bash && \
	sudo rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN pyenv install 3.9.12 && pyenv global 3.9.12

RUN pip3 install yq

RUN export GOARCH=$(go env GOARCH) && export GOOS=$(go env GOOS) && \
	mkdir /home/circleci/bin && \
	curl -sLo /home/circleci/bin/ocb \
	"https://github.com/open-telemetry/opentelemetry-collector/releases/download/v${OTC_BUILDER_VERSION}/ocb_${OTC_BUILDER_VERSION}_${GOOS}_${GOARCH}" && \
	chmod u+x /home/circleci/bin/ocb

RUN python --version && pip --version && yq --version && ocb version
