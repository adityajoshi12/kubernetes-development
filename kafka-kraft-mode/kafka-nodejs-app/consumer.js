require("dotenv").config();
const { Kafka } = require("kafkajs");
const kafka_address = process.env.BROKER;
const kafka = new Kafka({
  clientId: "my-app",
  brokers: [
   kafka_address
  ]
});
console.log(kafka_address);
const run = async () => {

  const consumer = kafka.consumer({ groupId: "test-group" });

  await consumer.connect();
  await consumer.subscribe({
    topic: "temperature",
    fromBeginning: true,
  });
  await consumer.run({
    eachMessage: async ({ topic, partition, message }) => {
      console.log({
        partition,
        offset: message.offset,
        value: message.value.toString(),
      });
    },
  });
};
run();
