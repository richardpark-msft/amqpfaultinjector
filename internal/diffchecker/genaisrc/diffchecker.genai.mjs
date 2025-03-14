import fs from 'fs';

// NOTE FOR USER TO RUN THIS SCRIPT:
// > cd internal/diffchecker/genaisrc
// > npx genaiscript run diffchecker.genai.mjs

// Read the prompt from the markdown file
const prompt_file = 'prompts/amqpdiff.prompt.md';
const prompt = fs.readFileSync(prompt_file, 'utf-8');
const log_py = 'amqpproxy-traffic-python.jsonl';
const log_net = 'amqpproxy-traffic-net.jsonl';

script({
    model: "azure:gpt-4o",
})

$`${prompt}`
def("NET", fs.readFileSync(log_net, 'utf-8'))
def("PYTHON", fs.readFileSync(log_py, 'utf-8'))
//$`You are an expert at the AMQP v1.0 protocol (https://www.amqp.org/sites/amqp.org/files/amqp.pdf).
//Identify the main operation that is being performed in the NET file and the PYTHON file. Examples of operations
//are send and receive. The most important frames to look at to determine the operation are the frames that come in after CBS authentication has been established.`

$`${prompt}`.cacheControl("ephemeral")
//defDiff("DIFF", log_py, log_net)
