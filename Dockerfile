FROM ubuntu:16.04

ADD /rancher-configurator /

CMD [ "/rancher-configurator" ]
