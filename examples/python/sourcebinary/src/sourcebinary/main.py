import sourcelibrary
import requests


def main():
    print(sourcelibrary.greet("World"))
    print(requests.get("http://example.com").status_code)
