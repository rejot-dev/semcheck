import time
from sentence_transformers import SentenceTransformer


def benchmark_model(model_name, sentences):
    """Benchmark a single embedding model."""
    print(f"\n{'='*50}")
    print(f"Benchmarking: {model_name}")
    print(f"{'='*50}")
    
    # Measure load time
    start_time = time.time()
    model = SentenceTransformer(model_name)
    load_time = time.time() - start_time
    print(f"Load time: {load_time:.3f} seconds")
    
    # Measure inference time
    start_time = time.time()
    embeddings = model.encode(sentences)
    inference_time = time.time() - start_time
    print(f"Inference time: {inference_time*1000:.1f} ms")
    print(f"Time per sentence: {inference_time*1000/len(sentences):.2f} ms")
    print(f"Embedding shape: {embeddings.shape}")
    
    return {
        'model': model_name,
        'load_time': load_time,
        'inference_time': inference_time,
        'time_per_sentence': inference_time/len(sentences),
        'embedding_shape': embeddings.shape
    }


def main():
    # Test sentences - 20 diverse English sentences
    test_sentences = [
        "The weather is lovely today and perfect for a walk.",
        "Machine learning algorithms are transforming modern technology.",
        "She enjoys reading mystery novels on quiet Sunday afternoons.",
        "The stock market experienced significant volatility this week.",
        "Children laughed and played in the sunny playground.",
        "The new restaurant serves authentic Italian cuisine.",
        "Climate change poses serious challenges for future generations.",
        "The concert was an incredible experience with amazing acoustics.",
        "Scientists discovered a new species of deep-sea creatures.",
        "The project deadline has been moved to next Friday.",
        "His dedication to excellence is truly inspiring to everyone.",
        "The ancient castle stood majestically on the hilltop.",
        "Technology companies are investing heavily in artificial intelligence.",
        "The garden bloomed with colorful flowers in springtime.",
        "International trade agreements affect global economic growth.",
        "The movie received outstanding reviews from critics worldwide.",
        "Space exploration continues to push the boundaries of human knowledge.",
        "The team celebrated their victory with great enthusiasm.",
        "Renewable energy sources are becoming increasingly cost-effective.",
        "The library provides a quiet sanctuary for focused study."
    ]
    
    print(f"Running benchmark with {len(test_sentences)} test sentences")
    
    # Models to benchmark
    models = [
        "Qwen/Qwen3-Embedding-0.6B",
        "intfloat/e5-small-v2",
        "BAAI/bge-small-en-v1.5",
        "avsolatorio/GIST-all-MiniLM-L6-v2",
        "Mihaiii/gte-micro-v4"
    ]
    
    results = []
    
    for model_name in models:
        try:
            result = benchmark_model(model_name, test_sentences)
            results.append(result)
        except Exception as e:
            print(f"Error benchmarking {model_name}: {e}")
    
    # Print summary
    print(f"\n{'='*50}")
    print("BENCHMARK SUMMARY")
    print(f"{'='*50}")
    
    if results:
        print(f"{'Model':<30} {'Load Time':<12} {'Inference':<12} {'Per Sentence':<15}")
        print("-" * 70)
        for result in results:
            print(f"{result['model']:<30} {result['load_time']:.3f}s{'':<6} {result['inference_time']*1000:.1f}ms{'':<6} {result['time_per_sentence']*1000:.2f}ms")
        
        # Find fastest model
        fastest_load = min(results, key=lambda x: x['load_time'])
        fastest_inference = min(results, key=lambda x: x['inference_time'])
        
        print(f"\nFastest load time: {fastest_load['model']}")
        print(f"Fastest inference: {fastest_inference['model']}")


if __name__ == "__main__":
    main()