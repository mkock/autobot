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

## Configuration

Autobot talks to several external systems and thefore require some configuration. Autobot configuration is provided
via a TOML file, which controls aspects of FTP connectivity, memory store integration, the actual synchronization
algorithm etc.

## Cache Internals

Redis / Google Memory Store is used for caching of vehicles. Vehicle lookups are supported via VIN and registration
numbers. In Redis, the lookup happens via two secondary indexes, which are sorted sets. Here's an overview:

- `autobot_vehicles` is a hashmap where the key is a hash of the Vehicle struct, metadata excluded, and the value
  is a serialized representation of Vehicle. Concretely, serialization is performed by converting the struct to JSON
  and then stringifying it.
- `autobot_vin_index` is a sorted set with keys following the pattern `<vin>:<hash>`. It acts as a lexicographical
  index with support for direct or partial VIN number lookups.
- `autobot_regnr_index` is a sorted set with keys following the pattern `<regnr>:<hash>`. It acts as a lexicographical
  index with support for direct or partial registration number lookups. Registration numbers are stored in uppercase
  which also requires searches to be performed with uppercase letters.

While the data structures are set in stone, their names are configurable via the config file.

## The Vehicle Lookup Mechanism

The following is a concrete explanation of how the Redis lookup mechanism works in Autobot.

Registration number lookups are performed against the sorted set `autobot_regnr_index`. For the license plate "AB13513",
Autobot performs the equivalent of the following command:

`ZRANGEBYLEX autobot_regnr_index "[AB13513" "[AB13513\xff"`

and similar for VIN lookups:

`ZRANGEBYLEX autobot_vin_index "[ZAR93200001204085" "[ZAR93200001204085\xff"`

This will return the vehicle hash, which can then be used to retrieve the vehicle directly:

`hget autobot_vehicles 16029023328557318062`

That's all there is to it.

## TODO

1. ~~Figure out how to efficiently perform REGNO+VIN lookups in Redis~~ Done
2. Remove values from indexes where the hash no longer refers to a vehicle
3. Add a CLI command that generates an empty config file for the sake of convenience
4. Add a CLI command for retrieving vehicle data, also for the sake of convenience
5. Add usage information when called without arguments

## Changelog

- _v1.0_ (Oct. 18) Initial version, with synchronization from DMR using Redis/Google Memory Store and data hashing.

## FAQ

_Why is it called Autobot?_

Because I like Transformers, and since we're dealing with cars here, the name was evident. Also, the synchronization
service qualifies as a bot :-)

_Why did you write this in Golang?_

Because Go is perfect for cloud applications such as this one. Also, I'm learning the language and was looking for a
real-world project to apply my knowledge to. So it's a learning project too.

_Why is this open source?_

It isn't, not yet.
