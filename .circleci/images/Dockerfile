FROM cimg/go:1.16

ENV PYENV_ROOT=/home/circleci/.pyenv \
	PATH=/home/circleci/.pyenv/shims:/home/circleci/.pyenv/bin:/home/circleci/.poetry/bin:$PATH

RUN sudo apt-get update && \
	sudo apt-get install -y \
		python \
		libbz2-dev \
		libreadline-dev \
		libssl-dev \
		python-openssl && \
	curl https://pyenv.run | bash && \
	sudo rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN pyenv install 3.9.6 && pyenv global 3.9.6

RUN pip3 install yq

RUN mkdir /home/circleci/bin && \
        wget https://github.com/open-telemetry/opentelemetry-collector-builder/releases/download/v0.30.0/opentelemetry-collector-builder_0.30.0_linux_amd64 \
	        -O /home/circleci/bin/opentelemetry-collector-builder && \
	sudo chmod u+x /home/circleci/bin/opentelemetry-collector-builder

RUN python --version && pip --version && yq --version && opentelemetry-collector-builder version
