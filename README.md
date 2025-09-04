# ha-myenecle

Unofficial add-on for Enecle (エネクル) gas monitoring service.

## Features

- Monitors Enecle gas usage.
- Runs a simple HTTP server on port 8000.
- Supports multiple architectures: aarch64, amd64, armhf, armv7, i386.

## Installation

1. Clone this repository:
   ```sh
   git clone https://github.com/yourusername/ha-myenecle.git
   ```

2. Build the Docker image:
   ```sh
   docker build --build-arg BUILD_FROM=python:3.10-slim -t ha-myenecle .
   ```

3. Run the container:
   ```sh
   docker run -p 8000:8000 ha-myenecle
   ```

## Usage

After starting, access the HTTP server at [http://localhost:8000](http://localhost:8000).

## Configuration

See `config.yaml` for available options.

## License

Unofficial project. Not affiliated