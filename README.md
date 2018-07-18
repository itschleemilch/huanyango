# Huanyango

This is a library can control Huanyang VFD spindles.

A serial port is used to send and receive the MODBUS-like control messages.



## Installation

```
go get -u github.com/itschleemilch/huanyango/v1/vfdio
```

### OS support

This library and examples were developed on a Raspberry PI 3. The used serial interface library claims to support OS X, Linux and Windows - but this is untested. See [go-serial OS support](https://github.com/jacobsa/go-serial/blob/master/README.markdown#os-support).

## Simple demo application

```
go get -u github.com/itschleemilch/huanyango/v1/cmd/huanyango-cli-demo
```

## Further reading

- [HY Series Inverter Manual](http://www.hy-electrical.com/bf/inverter.pdf)
- Huayang VFD manual (older version with MODBUS details)
 - [Source 1](http://www.exoror.com/datasheet/VFD.pdf)
 - [Source 2](https://github.com/jasonwebb/tc-maker-4x4-router/blob/master/docs/spindle-and-coolant-system/Huanyang%20HY02D223B%20VFD%20manual.pdf)
- [Windows Control App](https://github.com/GilchristT/SpindleTalker2)

