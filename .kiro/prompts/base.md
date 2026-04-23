# YOUR ROLE
You are a backend software engineer with extensive experience in low-level software development and protocol creation with mastery over the TCP protocol, DDD and Industry 4.0.

# Context
In Industry 4.0, the MQTT protocol is widely used for communication between IoT devices and systems. That is, machines publish data to an MQTT broker that is consumed and used by other systems. However, brokers typically do not offer ACL in open-source versions, robust queues, requiring paid versions for fine-grained control and data security.

# The Project
The idea is to create a micro MQTT broker FROM SCRATCH, from native TCP, to be used by a single machine with no dependency on fine-grained external access. That is, this should be an edge computing level application that must be an MQTT broker, which accepts only one client via MQTT, has the capacity to receive and store large amounts of data from this client with parallel and concurrent processes, ensuring resilience and the order of receipt of each piece of data, as well as the ingestion and recording of each one either on disk or database, so that after ingesting and guaranteeing, it should forward each piece of data to some external address, whether via Kafka broker, REST API, RMQ, S3, in short, over time we will have plugins that will handle this part. This is an Industry 4.0 project and should be built in a scalable manner to obtain OEE calculations, production orders in execution, order listing, piece counting, etc.

# Features
- Secure native connection: we must create the MQTT protocol from scratch via TCP, in a robust manner so that any machine or client (some Node-RED, Node application using some MQTT lib, or any device) can connect through username and password. The client must be able to connect and stay connected in a stable manner until it disconnects on its own. There must be a maximum of 5 clients connected at the same time, ensuring that clients can disconnect. If a sixth client tries to connect, it must be ignored.

- Topics: the broker must be able to accept data only on topics defined by environment variable. That is, the user via environment variable specifies the topics that the broker should accept. Clients can ONLY publish to the topics. It must be possible to define a maximum of 5 topics per broker.

- Data ingestion: for each topic, we must have a subscriber that will ingest each piece of data into a native FIFO queue in Go. Therefore, with a maximum of 5 topics, we will have a maximum of 5 queues.

- Queue flow: in each queue, we must save the newly arrived data in a SQLite database by locking the table and ensuring that this data was saved. The table should be called raw_data with the columns: client, topic, timezone, timestamp and payload (JSON), and only after being saved is the next piece of data consumed from the queue. Since we will have 5 queues, the lock on the table in a transaction is very important to ensure total performance and data consistency. After each piece of data is saved, we must forward this data to a new channel, which can have multiple workers.

- Workers: first we must think of workers as a scalable feature, which is why we receive data from a channel and forward it to multiple workers at once. Think that after the data is saved in the database we can already send it to an external S3, or Kafka, or evolve in the future to do some calculation and put it on a WebSocket channel, for example: calculate OEE of a machine, piece counting, production order, etc.

# Architecture
The architecture of this project must be as scalable as possible, always with contracts by interface and modules for each feature. Since it is not a common enterprise application, we must find the middle ground between hexagonal architecture, clean arch, modular architectures and what fits well in our case. We must think of everything in a scalable, decoupled manner, so that it is highly testable and easy to maintain. The project must be entirely built in Go, exploring all native language libraries to the fullest and avoiding external libs.

# References
You should consult the official repositories of existing open-source brokers such as eclipse-mosquitto, to understand and base yourself on the low level of the TCP protocol. Furthermore, you can get ideas on how to handle topics, etc., or whatever is only pertinent to the cases of this project.

# Your Objective
Initially, your objective is to create one or more documents in which you will investigate, understand (from the references), plan and give code examples for this project. This will be our starting point and you must be very thorough, cautious, detailed and unhurried, as it will be the foundation. Initially you should focus only on the # Features described. You should also create a document suggesting an architecture that makes sense for this project based on # Architecture. Don't forget to consult your skills in **.kiro/skills** so we have correct Go patterns and efficient testing patterns. You should also think about domains for this project, if it makes sense.
