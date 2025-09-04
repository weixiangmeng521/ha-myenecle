import http.server
import socketserver

PORT = 8444

class Handler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/":
            self.send_response(200)
            self.send_header("Content-type", "text/plain; charset=utf-8")
            self.end_headers()
            self.wfile.write(b"Hello from Enecle Home Assistant add-on!")
        else:
            self.send_error(404, "Not Found")

if __name__ == "__main__":
    with socketserver.TCPServer(("", PORT), Handler) as httpd:
        print(f"Serving Enecle add-on at http://0.0.0.0:{PORT}")
        httpd.serve_forever()