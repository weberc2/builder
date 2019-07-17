import sourcelibrary
import requests

def main():
    sourcelibrary.greet()
    print(requests.get("http://example.com").status_code)