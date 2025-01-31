# TextTrove

An application designed to provide Retrieval-Augmented Generation (RAG) interactions over a folder of markdown documents with minimal external dependencies.

## Overview

TextTrove is a fully functional application developed to enable interactive chats with Obsidian notes while offering a sandbox for enhancing my understanding of Large Language Models (LLM) and Retrieval-Augmented Generation (RAG). The primary goal was to create a locally running application that does not send data to cloud services, making ollama an ideal choice. The app uses an embedded vector store for embeddings, ensuring that everything remains self-contained on your local machine.

## How It Works

The application monitors your documents folder, parsing all markdown files and splitting them into smaller chunks. These chunks are then embedded and stored in an embedded vector store. When you ask a question, it’s embedded and a similarity search is performed against the vector store. The top five relevant documents are retrieved and used as context in the conversation with the LLM. Your query is then processed by the LLM as usual.

## Requirements

To use this app, you'll need:

- ollama running locally on the default interface and port (`localhost:11434`)
- A model for conversation (the app targets `llama3.2:latest` by default, but you can override this by setting `MODEL_CONVERSATION`)
- A model for embedding (the app targets `mxbai-embed-large:latest` by default; you can override with `MODEL_EMBEDDING_NAME`)
- A folder of markdown files
- The ability to run `go run main.go`

## Configuration

The app uses `envconfig` for configuration. Users should refer to `main.go` for a list of configurable items. For more information about `envconfig`, you can visit its [GitHub repository](https://github.com/kelseyhightower/envconfig).

## Usage

```sh
DOCUMENT_PATH=your/doc/folder go run main.go
```

### Customizable Prompts

The app uses templates for system and context prompts. You can customize these by dropping `system.tpl` and `context.tpl` in the `./prompts/` directory relative to the binary.

Refer to [chat.go](./pkg/models/chat.go) for the base templates.

## Features

- Interactive chat with LLM
- Retrieval of relevant documents based on query
- Local storage of embeddings
- Automatic parsing of markdown files
- Live-updating of document changes (watches for file modifications)
- Customizable system and context prompts (drop `system.tpl` and `context.tpl` in `./prompts/` relative to binary)

## Goals

- Enable interactive conversations with local markdown notes
- Maintain data privacy by running entirely on a local machine
- Provide a platform for learning and experimenting with LLM and RAG concepts

## Future Enhancements

- Allow configuration of custom ollama endpoints
- Tabbed interface for multiple concurrent conversations
- Copy/paste functionality for chat history
- Use tools to allow LLM to request further context if required
