from setuptools import setup, find_packages

setup(
    name="ggid-sdk",
    version="1.0.0",
    description="GGID IAM Platform Python SDK",
    packages=find_packages(),
    install_requires=[
        "requests>=2.28",
        "PyJWT>=2.8",
        "cryptography>=41.0",
    ],
    python_requires=">=3.8",
)
