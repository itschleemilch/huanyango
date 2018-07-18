# Huanyango

This is a library can control Huanyang VFD spindles.

A serial port is used to send and receive the MODBUS-like control messages.

## Installation

```
go get -u github.com/itschleemilch/huanyango/v1/vfdio
```

## Setup

```
                    +----------------------+                      +--------------------+
                    |                     B+---.  .---..---.  .---+ RS-                |
+-------------+     |      RS485           |    \/    /\    \/    |       Huanyang VFD |
| RPI 3    USB+-----+ Serial Interface    A+----..---.  .---..----+ RS+                |
+-------------+     |                      |                      +--------------------+
                    |                   GND+--------------------
                    +----------------------+
                                             Shielded Twisted Pair
                                             (GND single ended)
```

Required VFD settings:

```
# Enable RS485 Interface:
PD163 Communication Addresses   := 1

# Set baud rate to 9600 baud
PD164 Communication Baud Rate   := 1

# Set 8 data bytes, no party, one stop bit, RTU mode:
PD165 Communication Data Method := 3
```


### OS support

This library and examples were developed on a Raspberry PI 3. The used serial interface library claims to support OS X, Linux and Windows - but this is untested. See [go-serial OS support](https://github.com/jacobsa/go-serial/blob/master/README.markdown#os-support).

## Simple demo application

```
go get -u github.com/itschleemilch/huanyango/v1/cmd/huanyango-cli-demo
```
Example usage:

```
Huanyango Command Line Interface Demo
M3
S100
M5
M03
M4
?
Output RPM 1/min:  99
$
HyInverter Connected and Online:  true
```

## Further reading

1. [HY Series Inverter Manual](http://www.hy-electrical.com/bf/inverter.pdf)
2. Huayang VFD manual (older version with MODBUS details)
  - [Source 1](http://www.exoror.com/datasheet/VFD.pdf)
  - [Source 2](https://github.com/jasonwebb/tc-maker-4x4-router/blob/master/docs/spindle-and-coolant-system/Huanyang%20HY02D223B%20VFD%20manual.pdf)
3. [Windows Control App](https://github.com/GilchristT/SpindleTalker2)

