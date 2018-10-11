# Autobot

This is a microservice that provides a small API for looking up vehicle data by license plate and VIN number.
The purpose of this service, which was developed for OmniCar A/S, is to provide a caching mechanism for vehicle
lookups, which can be expensive as existing providers often charge a fee per lookup.

To further reduce the need for external lookups, a data synchronization service is included, which is currently
integrated with DMR, the Danish Motor vehicle Registry. Others can be added fairly easily.

## Purpose

The purpose is to store and provide easy access to vehicle history and registration information, using the following
key data for the core service:

- License plate number
- VIN
- Brand name
- Model name
- Fuel-type name
- First registration date
- Body type (if available)

## Technology Stack

- The microservice itself is written in Golang, v1.11
- Data is kept in a memory store: Redis for local development and Google Memory Store when deployed
- On a more detailed level, TOML is used for configuration files, FTP for DMR integration - others?

## FAQ

_Why is it called Autobot?_

Because I like Transformers, and since we're dealing with cars here, the name was evident. Also, the synchronization
service qualifies as a bot :-)

_Why did you write this in Golang?_

Because Go is perfect for cloud applications such as this one. Also, I'm learning the language and was looking for a
real-world project to apply my knowledge to. So it's a learning project too.

_Why is this open source?_

It isn't, not yet.
