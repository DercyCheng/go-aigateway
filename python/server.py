import argparse
import json
import time
import logging
from typing import List, Dict, Any, Optional
import os
import sys

import torch
from flask import Flask, request, jsonify
from transformers import AutoTokenizer, AutoModelForCausalLM, pipeline
import openai

from error_handling import (
    handle_errors, validate_request_data, rate_limit, require_api_key,
    log_request, with_resource_management, resource_manager,
    ValidationError, SecurityError, ResourceError
)

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s - %(pathname)s:%(lineno)d'
)
logger = logging.getLogger(__name__)

app = Flask(__name__)

# Global variables
model = None
tokenizer = None
embedding_model = None
model_type = "chat"
model_size = "small"
third_party_client = None
use_third_party = False

# Model selection based on size
MODEL_MAP = {
    "small": {
        "chat": "TinyLlama/TinyLlama-1.1B-Chat-v1.0",
        "completion": "microsoft/phi-2",
        "embedding": "sentence-transformers/all-MiniLM-L6-v2"
    },
    "medium": {
        "chat": "TinyLlama/TinyLlama-1.1B-Chat-v1.0",
        "completion": "microsoft/phi-2",
        "embedding": "sentence-transformers/all-MiniLM-L6-v2" 
    },
    "large": {
        "chat": "HuggingFaceH4/mistral-7b-instruct-v0.2",
        "completion": "google/gemma-2b",
        "embedding": "intfloat/e5-large-v2"
    }
}

# Third-party model configurations
THIRD_PARTY_MODELS = {
    "dashscope": {
        "base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
        "models": {
            "chat": ["qwen-turbo", "qwen-plus", "qwen-max", "qwen-max-1201", "qwen-max-longcontext"],
            "completion": ["text-davinci-003", "text-curie-001"],
            "embedding": ["text-embedding-v1", "text-embedding-v2"]
        }
    }
}

def initialize_model():
    global model, tokenizer, embedding_model, model_type, model_size, third_party_client, use_third_party
    
    # Setup Hugging Face mirror for China mainland
    if os.getenv('HF_ENDPOINT'):
        os.environ['HF_ENDPOINT'] = os.getenv('HF_ENDPOINT')
        logger.info(f"Using Hugging Face mirror: {os.getenv('HF_ENDPOINT')}")
    
    # Check if we should use third-party models
    if os.getenv('USE_THIRD_PARTY_MODEL', '').lower() == 'true':
        initialize_third_party_model()
        return
    
    model_id = MODEL_MAP[model_size][model_type]
    logger.info(f"Initializing {model_type} model: {model_id}")
    
    device = "cuda" if torch.cuda.is_available() else "cpu"
    logger.info(f"Using device: {device}")
    
    try:
        if model_type in ["chat", "completion"]:
            logger.info(f"Downloading tokenizer for: {model_id}")
            tokenizer = AutoTokenizer.from_pretrained(
                model_id,
                cache_dir=os.getenv('TRANSFORMERS_CACHE', None),
                trust_remote_code=True
            )
            logger.info(f"Downloading model for: {model_id}")
            model = AutoModelForCausalLM.from_pretrained(
                model_id,
                torch_dtype=torch.float16 if device == "cuda" else torch.float32,
                low_cpu_mem_usage=True,
                device_map=device,
                cache_dir=os.getenv('TRANSFORMERS_CACHE', None),
                trust_remote_code=True
            )
        
        if model_type == "embedding" or model_size == "large":
            embedding_model_id = MODEL_MAP[model_size]["embedding"]
            logger.info(f"Initializing embedding model: {embedding_model_id}")
            embedding_model = pipeline(
                "feature-extraction", 
                model=embedding_model_id, 
                device=device,
                model_kwargs={
                    "cache_dir": os.getenv('TRANSFORMERS_CACHE', None),
                    "trust_remote_code": True
                }
            )
    except Exception as e:
        logger.error(f"Error downloading/loading model: {str(e)}")
        logger.info("This might be due to network issues. Please check your internet connection and Hugging Face mirror configuration.")
        raise

def initialize_third_party_model():
    global third_party_client, use_third_party
    
    # Get configuration from environment variables
    api_key = os.getenv('BAILIAN_API_KEY')
    if not api_key:
        raise ValueError("BAILIAN_API_KEY environment variable is required for third-party models")
    
    base_url = THIRD_PARTY_MODELS["dashscope"]["base_url"]
    
    # Initialize OpenAI client with 阿里百炼 configuration
    third_party_client = openai.OpenAI(
        api_key=api_key,
        base_url=base_url
    )
    
    use_third_party = True
    logger.info(f"Initialized 阿里百炼 third-party model client with base URL: {base_url}")

def get_available_models():
    """Get list of available models based on current configuration"""
    if use_third_party:
        models = []
        for model_type, model_list in THIRD_PARTY_MODELS["dashscope"]["models"].items():
            for model_name in model_list:
                models.append({
                    "id": model_name,
                    "object": "model",
                    "created": int(time.time()) - 10000,
                    "owned_by": "alililian"
                })
        return models
    else:
        models = [
            {
                "id": MODEL_MAP["small"]["chat"],
                "object": "model",
                "created": int(time.time()) - 10000,
                "owned_by": "local"
            },
            {
                "id": MODEL_MAP["small"]["completion"],
                "object": "model",
                "created": int(time.time()) - 10000,
                "owned_by": "local"
            },
            {
                "id": MODEL_MAP["small"]["embedding"],
                "object": "model",
                "created": int(time.time()) - 10000,
                "owned_by": "local"
            }
        ]
        
        if model_size != "small":
            for model_type in ["chat", "completion", "embedding"]:
                models.append({
                    "id": MODEL_MAP[model_size][model_type],
                    "object": "model",
                    "created": int(time.time()) - 10000,
                    "owned_by": "local"
                })
        
        return models

@app.route('/health', methods=['GET'])
@handle_errors
@log_request
def health_check():
    """Enhanced health check with resource status"""
    try:
        status = resource_manager.get_status()
        status.update({
            "status": "healthy",
            "timestamp": int(time.time()),
            "model_loaded": model is not None or use_third_party,
            "model_type": model_type,
            "model_size": model_size,
            "third_party_enabled": use_third_party
        })
        
        # Check model health
        if not use_third_party and model is None:
            status["status"] = "degraded"
            status["issues"] = ["Model not loaded"]
        
        if use_third_party and third_party_client is None:
            status["status"] = "degraded"
            status["issues"] = ["Third-party client not initialized"]
        
        # Check resource constraints
        if status.get("active_requests", 0) >= resource_manager.max_concurrent_requests:
            status["status"] = "busy"
            status["issues"] = status.get("issues", []) + ["At capacity"]
        
        return jsonify(status), 200
    except Exception as e:
        logger.error(f"Health check failed: {str(e)}")
        return jsonify({
            "status": "unhealthy",
            "timestamp": int(time.time()),
            "error": str(e)
        }), 503

@app.route('/v1/chat/completions', methods=['POST'])
@handle_errors
@log_request
@rate_limit(max_requests=30, window=60)
@validate_request_data(required_fields=['messages'])
@with_resource_management
def chat_completions():
    """Enhanced chat completions with comprehensive validation and error handling"""
    if not use_third_party and (model is None or tokenizer is None):
        raise ResourceError("Model not loaded", "model")
    
    if use_third_party and third_party_client is None:
        raise ResourceError("Third-party client not initialized", "client")
    
    try:
        data = request.get_json()
        messages = data.get('messages', [])
        max_tokens = data.get('max_tokens', 1024)
        temperature = data.get('temperature', 0.7)
        model_name = data.get('model', 'qwen-turbo' if use_third_party else MODEL_MAP[model_size][model_type])
        
        # Validate parameters
        if not isinstance(messages, list) or len(messages) == 0:
            raise ValidationError("Messages must be a non-empty array", "messages")
        
        if not (1 <= max_tokens <= 4096):
            raise ValidationError("max_tokens must be between 1 and 4096", "max_tokens")
        
        if not (0.0 <= temperature <= 2.0):
            raise ValidationError("temperature must be between 0.0 and 2.0", "temperature")
        
        # Validate message format
        for i, msg in enumerate(messages):
            if not isinstance(msg, dict):
                raise ValidationError(f"Message {i} must be an object", f"messages[{i}]")
            
            if 'role' not in msg or 'content' not in msg:
                raise ValidationError(f"Message {i} missing role or content", f"messages[{i}]")
            
            if msg['role'] not in ['system', 'user', 'assistant']:
                raise ValidationError(f"Invalid role in message {i}", f"messages[{i}].role")
            
            if not isinstance(msg['content'], str):
                raise ValidationError(f"Content in message {i} must be string", f"messages[{i}].content")
            
            if len(msg['content']) > 8000:
                raise ValidationError(f"Content in message {i} too long", f"messages[{i}].content")
        
        logger.info(f"Processing chat completion: {len(messages)} messages, max_tokens={max_tokens}, third_party={use_third_party}")
        
        # Handle third-party models
        if use_third_party:
            start_time = time.time()
            try:
                response = third_party_client.chat.completions.create(
                    model=model_name,
                    messages=messages,
                    max_tokens=max_tokens,
                    temperature=temperature
                )
                
                generation_time = time.time() - start_time
                logger.info(f"Third-party chat completion successful in {generation_time:.2f}s")
                
                # Convert to our standard format
                return jsonify({
                    "id": response.id,
                    "object": "chat.completion",
                    "created": response.created,
                    "model": response.model,
                    "system_fingerprint": "alililian-third-party",
                    "choices": [
                        {
                            "index": choice.index,
                            "message": {
                                "role": choice.message.role,
                                "content": choice.message.content
                            },
                            "finish_reason": choice.finish_reason
                        } for choice in response.choices
                    ],
                    "usage": {
                        "prompt_tokens": response.usage.prompt_tokens,
                        "completion_tokens": response.usage.completion_tokens,
                        "total_tokens": response.usage.total_tokens
                    }
                })
                
            except Exception as e:
                logger.error(f"Third-party API error: {str(e)}")
                raise ResourceError(f"Third-party model error: {str(e)}", "third_party_api")
        
        # Handle local models (existing code)
        # Format the conversation for the model
        prompt = ""
        for msg in messages:
            role = msg['role']
            content = msg['content']
            if role == 'system':
                prompt += f"<|system|>\n{content}\n"
            elif role == 'user':
                prompt += f"<|user|>\n{content}\n"
            elif role == 'assistant':
                prompt += f"<|assistant|>\n{content}\n"
        
        prompt += "<|assistant|>\n"
        
        # Tokenize and check length
        inputs = tokenizer(prompt, return_tensors="pt", truncation=True, max_length=2048)
        if inputs["input_ids"].shape[1] > 2048:
            raise ValidationError("Input too long after tokenization", "messages")
        
        inputs = inputs.to(model.device)
        
        # Generate response with timeout protection
        start_time = time.time()
        with torch.no_grad():
            outputs = model.generate(
                inputs["input_ids"],
                max_new_tokens=min(max_tokens, 1024),  # Cap max tokens
                temperature=temperature,
                do_sample=temperature > 0,
                pad_token_id=tokenizer.eos_token_id,
                attention_mask=inputs.get("attention_mask"),
                use_cache=True
            )
        
        generation_time = time.time() - start_time
        if generation_time > 30:  # 30 second timeout
            logger.warning(f"Generation took {generation_time:.2f}s, might be too slow")
        
        response_text = tokenizer.decode(
            outputs[0][inputs["input_ids"].shape[1]:], 
            skip_special_tokens=True
        )
        
        # Create response
        response = {
            "id": f"chatcmpl-{int(time.time() * 1000)}",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": model_name,
            "system_fingerprint": "local-python-model",
            "choices": [
                {
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": response_text.strip()
                    },
                    "finish_reason": "stop"
                }
            ],
            "usage": {
                "prompt_tokens": inputs["input_ids"].shape[1],
                "completion_tokens": outputs.shape[1] - inputs["input_ids"].shape[1],
                "total_tokens": outputs.shape[1]
            }
        }
        
        logger.info(f"Chat completion successful: {response['usage']['total_tokens']} tokens in {generation_time:.2f}s")
        return jsonify(response)
        
    except torch.cuda.OutOfMemoryError:
        torch.cuda.empty_cache()  # Clear GPU cache
        raise ResourceError("GPU out of memory", "gpu_memory")
    except Exception as e:
        logger.error(f"Error in chat completions: {str(e)}")
        raise

@app.route('/v1/completions', methods=['POST'])
@handle_errors
@log_request
@rate_limit(max_requests=30, window=60)
@validate_request_data(required_fields=['prompt'])
@with_resource_management
def completions():
    """Text completion endpoint with third-party support"""
    if not use_third_party and (model is None or tokenizer is None):
        raise ResourceError("Model not loaded", "model")
    
    if use_third_party and third_party_client is None:
        raise ResourceError("Third-party client not initialized", "client")
    
    try:
        data = request.json
        prompt = data.get('prompt', '')
        max_tokens = data.get('max_tokens', 1024)
        temperature = data.get('temperature', 0.7)
        model_name = data.get('model', 'text-davinci-003' if use_third_party else MODEL_MAP[model_size][model_type])
        
        logger.info(f"Completion request with prompt length: {len(prompt)}, third_party={use_third_party}")
        
        # Handle third-party models
        if use_third_party:
            start_time = time.time()
            try:
                response = third_party_client.completions.create(
                    model=model_name,
                    prompt=prompt,
                    max_tokens=max_tokens,
                    temperature=temperature
                )
                
                generation_time = time.time() - start_time
                logger.info(f"Third-party completion successful in {generation_time:.2f}s")
                
                return jsonify({
                    "id": response.id,
                    "object": "text_completion",
                    "created": response.created,
                    "model": response.model,
                    "choices": [
                        {
                            "text": choice.text,
                            "index": choice.index,
                            "finish_reason": choice.finish_reason
                        } for choice in response.choices
                    ],
                    "usage": {
                        "prompt_tokens": response.usage.prompt_tokens,
                        "completion_tokens": response.usage.completion_tokens,
                        "total_tokens": response.usage.total_tokens
                    }
                })
                
            except Exception as e:
                logger.error(f"Third-party completion API error: {str(e)}")
                raise ResourceError(f"Third-party model error: {str(e)}", "third_party_api")
        
        # Handle local models (existing code)
        inputs = tokenizer(prompt, return_tensors="pt").to(model.device)
        
        # Generate response
        outputs = model.generate(
            inputs["input_ids"],
            max_new_tokens=max_tokens,
            temperature=temperature,
            do_sample=temperature > 0,
        )
        
        response_text = tokenizer.decode(outputs[0][inputs["input_ids"].shape[1]:], skip_special_tokens=True)
        
        # Create response
        response = {
            "id": f"cmpl-{int(time.time())}",
            "object": "text_completion",
            "created": int(time.time()),
            "model": model_name,
            "choices": [
                {
                    "text": response_text.strip(),
                    "index": 0,
                    "finish_reason": "stop"
                }
            ],
            "usage": {
                "prompt_tokens": inputs["input_ids"].shape[1],
                "completion_tokens": outputs.shape[1] - inputs["input_ids"].shape[1],
                "total_tokens": outputs.shape[1]
            }
        }
        
        return jsonify(response)
    
    except Exception as e:
        logger.error(f"Error in completions: {str(e)}")
        raise

@app.route('/v1/embeddings', methods=['POST'])
@handle_errors
@log_request
@rate_limit(max_requests=30, window=60)
@validate_request_data(required_fields=['input'])
@with_resource_management
def embeddings():
    """Embeddings endpoint with third-party support"""
    if not use_third_party and embedding_model is None:
        raise ResourceError("Embedding model not initialized", "embedding_model")
    
    if use_third_party and third_party_client is None:
        raise ResourceError("Third-party client not initialized", "client")
        
    try:
        data = request.json
        input_texts = data.get('input', [])
        model_name = data.get('model', 'text-embedding-v1' if use_third_party else MODEL_MAP[model_size]["embedding"])
        
        if isinstance(input_texts, str):
            input_texts = [input_texts]
            
        logger.info(f"Embedding request with {len(input_texts)} inputs, third_party={use_third_party}")
        
        # Handle third-party models
        if use_third_party:
            start_time = time.time()
            try:
                response = third_party_client.embeddings.create(
                    model=model_name,
                    input=input_texts
                )
                
                generation_time = time.time() - start_time
                logger.info(f"Third-party embeddings successful in {generation_time:.2f}s")
                
                return jsonify({
                    "object": "list",
                    "data": [
                        {
                            "object": "embedding",
                            "embedding": embedding.embedding,
                            "index": embedding.index
                        } for embedding in response.data
                    ],
                    "model": response.model,
                    "usage": {
                        "prompt_tokens": response.usage.prompt_tokens,
                        "total_tokens": response.usage.total_tokens
                    }
                })
                
            except Exception as e:
                logger.error(f"Third-party embeddings API error: {str(e)}")
                raise ResourceError(f"Third-party model error: {str(e)}", "third_party_api")
        
        # Handle local models (existing code)
        # Generate embeddings
        embeddings = []
        token_count = 0
        
        for i, text in enumerate(input_texts):
            # Get embedding
            embedding_output = embedding_model(text)
            # Average across tokens to get a single vector per text
            embedding_vector = torch.mean(torch.tensor(embedding_output[0]), dim=0).tolist()
            
            embeddings.append({
                "object": "embedding",
                "embedding": embedding_vector,
                "index": i
            })
            
            # Approximate token count
            token_count += len(text.split())
        
        # Create response
        response = {
            "object": "list",
            "data": embeddings,
            "model": model_name,
            "usage": {
                "prompt_tokens": token_count,
                "total_tokens": token_count
            }
        }
        
        return jsonify(response)
    
    except Exception as e:
        logger.error(f"Error in embeddings: {str(e)}")
        raise

@app.route('/v1/models', methods=['GET'])
@handle_errors
@log_request
def list_models():
    """List available models endpoint"""
    try:
        models = get_available_models()
        response = {
            "object": "list",
            "data": models
        }
        return jsonify(response)
    except Exception as e:
        logger.error(f"Error listing models: {str(e)}")
        raise

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Local AI Model Server with Third-Party Support')
    parser.add_argument('--host', type=str, default='localhost', help='Host to bind the server to')
    parser.add_argument('--port', type=int, default=5000, help='Port to bind the server to')
    parser.add_argument('--model-type', type=str, default='chat', choices=['chat', 'completion', 'embedding'], help='Type of model to use')
    parser.add_argument('--model-size', type=str, default='small', choices=['small', 'medium', 'large'], help='Size of model to use')
    parser.add_argument('--use-third-party', action='store_true', help='Use third-party models (阿里百炼)')
    
    args = parser.parse_args()
    
    model_type = args.model_type
    model_size = args.model_size
    
    # Set environment variable for third-party usage
    if args.use_third_party:
        os.environ['USE_THIRD_PARTY_MODEL'] = 'true'
    
    logger.info(f"Starting server with model type: {model_type}, size: {model_size}, third_party: {args.use_third_party}")
    
    try:
        initialize_model()
        logger.info("Model initialization completed successfully")
    except Exception as e:
        logger.error(f"Failed to initialize model: {str(e)}")
        sys.exit(1)
    
    app.run(host=args.host, port=args.port)
