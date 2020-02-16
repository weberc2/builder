from unittest import TestCase, main

from sourcelibrary.greet import greet


class GreetTest(TestCase):
    def test_greet(self):
        self.assertEqual("Hello, World", greet("World"))


if __name__ == "__main__":
    main()
