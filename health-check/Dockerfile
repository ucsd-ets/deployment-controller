FROM python:3.10

WORKDIR /app
COPY health-check.py /app

# install pip packages
COPY requirements.txt requirements.txt
RUN pip3 install -r requirements.txt

CMD [ "python3", "health-check.py" ]
