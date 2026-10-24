# all-MiniLM-L6-v2 (float16)

Sentence-embedding model used by OFFLINE mode (step 4) to compare the meaning of column names and values. It is loaded by the pure-Go runner in [embedding/](../../embedding/), with no network access, and it can only turn text into vectors, so it cannot follow instructions.

- **Source:** [sentence-transformers/all-MiniLM-L6-v2](https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2), downloaded 2026-10-23
- **License:** Apache License 2.0 (see [LICENSE](LICENSE)). Copyright belongs to the model's authors; it is redistributed here unchanged except for the conversion below.
- **Language:** English

## Files

| File | Description | SHA-256 |
|---|---|---|
| `model.safetensors` | Weights converted from float32 to float16 (43 MB instead of 87 MB); unused pooler weights removed | `ef4f66adbac55a55f9d5a2d865d84a17f42e2f202b49d0235c77d3cdd5b19ce9` |
| `vocab.txt` | Tokenizer vocabulary, unchanged | `07eced375cec144d27c900241f3e339478dec958f92fddbc551f295c992038a3` |
| `config.json` | Model configuration, unchanged | `953f9c0d463486b10a6871cc2fd59f223b2c70184f49815e7efbcab5d8908b41` |

The original `model.safetensors` had SHA-256 `53aa51172d142c89d9012cce15ae4d6cc0ca6895895114379cacb4fab128d9db`.

Safetensors is a plain data format, so loading the model cannot execute code.

## Reproducing the conversion

```bash
curl -L -o original.safetensors https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/model.safetensors
go run tools/convert_model/main.go original.safetensors models/all-MiniLM-L6-v2/model.safetensors
```

## Accuracy check

Embeddings from the Go runner were compared with the model's official ONNX export (run with ONNX Runtime and the Hugging Face tokenizer) on 16 test strings, including accented and Chinese text. Token ids matched exactly, and the lowest cosine similarity was 0.9999999 with the original weights and 0.9999995 with these float16 weights.
