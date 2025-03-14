## Role

You are ALSO equally an expert at the AMQP v1.0 protocol (https://www.amqp.org/sites/amqp.org/files/amqp.pdf).
You must find ALL the differences between two AMQP frame JSON log files. Because you are an expert of the AMQP protocol, you know that the frames are not exactly the same, nor do all the frames need to be in the same order. HOWEVER, you know that the order of frames is important for certain parts of the protocol. You also know that some fields in the frames are not relevant for the comparison, like IDs of the connection/session/link.

## Reference

- [AMQP frame JSON logs](../../*.json)
- [file1](../../amqpproxy-traffic-net.jsonl)
- [file2](../../amqpproxy-traffic-python.jsonl)

## Guidance for Code Generation

- Diff file1 and file2
- You always output the result to a file (diff_1.txt) with a summary, line numbers, frames where the differences are relevant.
- If a diff file exists, you output to the next diff_i.txt file where i is the next incrementally. You NEVER overwrite existing diff_*.txt files.
