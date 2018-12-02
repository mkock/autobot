# Autobot

Autobot is a microservice that provides a small API for looking up vehicle data by license plate, VIN number and
possibly other vehicle master data.

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
- On a more detailed level, TOML is used for configuration files, FTP for DMR integration and Redis/Memory Store
  is the main vehicle store and indexing mechanism. The rest is just idiomatic Go :-)

## Configuration

Autobot talks to several external systems and thefore require some configuration. Autobot configuration is provided
via a TOML file, which controls aspects of FTP connectivity, memory store integration, the actual synchronization
algorithm etc.

## API

- `GET /` returns a simple status, ie. uptime etc.
- `GET /vehiclestore/status` returns a status for the vehicle store, ie. last sync time and number of vehicles.
- `GET /lookup` looks up a vehicle by hash value or a combination of country and registration- or VIN number.
- `PATCH /vehicle` disables/enables a vehicle by hash value.
- `PUT /vehicle` _(planned)_ updates a vehicle's master data

## Package Structure

- `config` - contains app configuration and a loader that reads configuration data from a local TOML file.
- `vehicle` - contains the Vehicle entity and related functions, plus the implementation of the vehicle store.
- `dataprovider` - contains abstractions and implementations for loading data from varying sources, currently ftp
  and the local file system.
- `dmr` - contains the integration with DMR, the Danish Motor Registry: parsers and data representations.
- `app` - the entrance to the application itself: command line parser and runner that will both execute CLI commands
  and control the webservice.
- `webservice` - this is the webservice part of the application which provides a REST-style HTTP API.
- `main` - application bootstrapping.

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

1. ~~Figure out how to efficiently perform REGNO+VIN lookups in Redis~~ _Done_
2. ~~Remove values from indexes where the hash no longer refers to a vehicle~~ _Done_
3. ~~Add a CLI command that generates an empty config file for the sake of convenience~~ _Done_
4. ~~Add a CLI command for retrieving vehicle data, also for the sake of convenience~~ _Done_
5. ~~Add usage information when called without arguments~~ _Done_
6. ~~Add support for disabling vehicles~~ _Done_
7. ~~Build a simple HTTP API with support for lookups~~ _Done_
8. Switch from Go's builtin http package to Gin and add request logging, central error handling etc.
9. ~~Allow the user to disable and re-enable vehicles via the API~~ _Done_
10. Allow the user to create revisions of vehicles via the API
11. Handle `*net.OpError` (network interruptions) during sync, if it makes sense
12. Implement a cleanup job that removes all vehicles from the store that are not present in an index
13. Add a discrete progress indicator while running sync (CLI only)
14. ~~Split up data providers and their configs so autobot will support multiple providers~~ _Done_
15. Consider providing optional CSV output for both CLI and API
16. Add a CLI command "test" that tests integration with each provider
17. Add support for direct vehicle lookups in case of cache misses?
18. Achieve some test coverage!
19. Improve quality of imported vehicles: ignore old (recycled plates) and invalid vehicle data

## Changelog

- _v1.0_ (Nov. 18) Initial version, with synchronization from DMR using Redis/Google Memory Store and data hashing.

## FAQ

_Why is it called Autobot?_

Because I like Transformers, and since we're dealing with cars here, the name was evident. Also, the synchronization
service qualifies as a bot :-)

_Why does Autobot exist?_

It exists to meet a concrete need at my workplace, OmniCar A/S. As a company that deals in service contracts on
vehicles, there is a regular need for performing lookups based on license plate numbers and other car master data.

These lookups are rarely free - in fact, the services that provide them often charge a fee per lookup. To save money
on lookups, it becomes prudent to not only cache the results, but also to create our own vehicle database if possible.

Autobot is the answer to these needs.

_Why did you write this in Golang?_

Because Go is perfect for cloud applications such as this one. Also, I'm learning the language and was looking for a
real-world project to apply my knowledge to. So it's a learning project too.

_Why is this open source?_

It isn't, not yet. But it aims to be.
