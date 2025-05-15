using Azure.Messaging.ServiceBus;
using Azure.Identity;
using System.Net;
using System.Security.Cryptography.X509Certificates;
using System.Net.Security;
using Azure.Messaging.EventHubs.Producer;

var env = File.ReadAllLines(".env")
    .Select(line => line.Split("=", 2))
    .Where(parts => parts.Length == 2);

ServicePointManager.ServerCertificateValidationCallback = (object sender, X509Certificate? certificate, X509Chain? chain, SslPolicyErrors errors) =>
{
    return true;
};

// Check for a command line argument to select which sample to run
if (args.Length > 0 && args[0].Equals("servicebus", StringComparison.OrdinalIgnoreCase))
{
    await ServiceBusSample();
}
else if (args.Length > 0 && args[0].Equals("eventhubs", StringComparison.OrdinalIgnoreCase))
{
    await EventHubsSample();
}
else
{
    Console.WriteLine("Usage: <program> [eventhubs|servicebus]");
}

async Task ServiceBusSample()
{
    Console.WriteLine("Starting Service Bus sample");

    var endpoint = env.First(pair => pair[0] == "SERVICEBUS_ENDPOINT")[1];
    var entity = env.First(pair => pair[0] == "SERVICEBUS_QUEUE")[1];
    var tokenCred = new DefaultAzureCredential();

    Console.WriteLine($"Running service bus test, connecting to {endpoint}, using queue {entity}");

    await using var client = new ServiceBusClient(endpoint, tokenCred, new ServiceBusClientOptions()
    {
        CertificateValidationCallback = (sender, certificate, chain, sslPolicyErrors) =>
            {
                return true;
            },
        CustomEndpointAddress = new Uri("sb://localhost:5671")
    });

    await using var sender = client.CreateSender(entity);
    await sender.SendMessageAsync(new ServiceBusMessage(BinaryData.FromString("hello world")));

    Console.WriteLine("Done, message sent!");
}

async Task EventHubsSample()
{
    Console.WriteLine("Starting Event Hubs sample");

    var endpoint = env.First(pair => pair[0] == "EVENTHUBS_ENDPOINT")[1];
    var entity = env.First(pair => pair[0] == "EVENTHUBS_HUBNAME")[1];
    var tokenCred = new DefaultAzureCredential();

    Console.WriteLine($"Running event hubs test, connecting to {endpoint}, using queue {entity}");

    await using var producer = new EventHubProducerClient(endpoint, entity, tokenCred, new EventHubProducerClientOptions()
    {
        ConnectionOptions = new Azure.Messaging.EventHubs.EventHubConnectionOptions()
        {
            CertificateValidationCallback = (sender, certificate, chain, sslPolicyErrors) =>
            {
                return true;
            },
            CustomEndpointAddress = new Uri("sb://localhost:5671"),
        }
    });

    Console.WriteLine($"Connecting to {endpoint}/{entity}");
    var batch = await producer.CreateBatchAsync();
    _ = batch.TryAdd(new Azure.Messaging.EventHubs.EventData("hello world"));

    Console.WriteLine("Sending message");
    await producer.SendAsync(batch);

    Console.WriteLine("Done");
}
