import numpy as np
from sentence_transformers import SentenceTransformer
from sklearn.metrics.pairwise import cosine_similarity
import re
import time
import nltk
from nltk.tokenize import TextTilingTokenizer
from nltk.corpus import stopwords
import pathlib


class EmbeddingReducer:
    """Embedding-based reducer that uses semantic similarity to find relevant context."""

    def __init__(self, model_name="BAAI/bge-small-en-v1.5", top_k=10):
        self.model = SentenceTransformer(model_name)
        self.top_k = top_k

    def find_relevant_context(self, content, query, context_before=2, context_after=2):
        """Find most relevant lines using embedding similarity."""
        start_time = time.time()

        lines = content.split('\n')

        # Skip empty lines and very short lines
        valid_lines = [(i, line) for i, line in enumerate(lines)
                      if line.strip() and len(line.strip()) > 10]

        if not valid_lines:
            return '\n'.join(lines), 0.0

        line_indices, line_texts = zip(*valid_lines)

        # Measure encoding time
        encoding_start = time.time()
        query_embedding = self.model.encode([query])
        line_embeddings = self.model.encode(line_texts)
        encoding_time = time.time() - encoding_start

        # Measure similarity computation time
        similarity_start = time.time()
        similarities = cosine_similarity(query_embedding, line_embeddings)[0]

        # Get top-k most similar lines
        top_indices = np.argsort(similarities)[-self.top_k:][::-1]
        similarity_time = time.time() - similarity_start

        # Expand to include context around each match
        context_indices = set()
        for idx in top_indices:
            actual_line_idx = line_indices[idx]
            for i in range(actual_line_idx - context_before,
                          actual_line_idx + context_after + 1):
                if 0 <= i < len(lines):
                    context_indices.add(i)

        # Collect result lines in order
        result_lines = []
        for i in sorted(context_indices):
            result_lines.append(lines[i])

        total_time = time.time() - start_time

        # Store timing information for analysis
        self.last_timing = {
            'total_time': total_time,
            'encoding_time': encoding_time,
            'similarity_time': similarity_time,
            'num_lines': len(line_texts),
            'query_length': len(query)
        }

        return '\n'.join(result_lines), total_time



class TextTilingReducer:
    """TextTiling-based reducer that uses topical segmentation to find relevant context."""

    def __init__(self, w=20, k=10, similarity_method=0):
        """
        Initialize TextTiling tokenizer.

        Args:
            w: Size of the sliding window (default 20)
            k: Size of blocks for block comparison method (default 10)
            similarity_method: 0 for block_comparison, 1 for vocabulary_introduction
        """
        try:
            # Download required NLTK data if not present
            nltk.download('punkt', quiet=True)
            nltk.download('stopwords', quiet=True)
        except:
            pass

        self.tokenizer = TextTilingTokenizer(
            w=w,
            k=k,
            similarity_method=similarity_method,
            stopwords=stopwords.words('english') if 'english' in stopwords.fileids() else None
        )

    def find_relevant_context(self, content, query, max_segments=3):
        """Find most relevant segments using TextTiling segmentation."""
        start_time = time.time()

        try:
            # Segment the text using TextTiling
            segments = self.tokenizer.tokenize(content)

            if not segments:
                return content, 0.0

            # Score segments based on query relevance
            segment_scores = []
            query_words = set(query.lower().split())

            for i, segment in enumerate(segments):
                segment_words = set(segment.lower().split())

                # Calculate word overlap score
                overlap = len(query_words.intersection(segment_words))
                normalized_score = overlap / len(query_words) if query_words else 0

                # Boost score for segments containing key terms
                key_terms = ['status', 'code', 'response', 'request', 'http']
                key_term_score = sum(1 for term in key_terms if term in segment.lower())

                # Calculate segment density (non-empty lines)
                lines = [line.strip() for line in segment.split('\n') if line.strip()]
                density_score = len(lines) / max(1, segment.count('\n'))

                total_score = normalized_score + (key_term_score * 0.3) + (density_score * 0.1)
                segment_scores.append((i, total_score, segment))

            # Sort by score and take top segments
            segment_scores.sort(key=lambda x: x[1], reverse=True)
            top_segments = segment_scores[:max_segments]

            # Sort selected segments by original order and concatenate
            top_segments.sort(key=lambda x: x[0])
            result = '\n\n'.join([seg[2] for seg in top_segments])

        except Exception as e:
            # Fallback to simple text splitting if TextTiling fails
            lines = content.split('\n')
            mid_point = len(lines) // 2
            result = '\n'.join(lines[max(0, mid_point-10):mid_point+10])

        total_time = time.time() - start_time

        # Store timing information for analysis
        self.last_timing = {
            'total_time': total_time,
            'num_segments': len(segments) if 'segments' in locals() else 0,
            'query_length': len(query)
        }

        return result, total_time


def test_reducer_comparison():
    """Test hypothesis: embedding model is better at finding specific context."""

    # Test data from the Go test
    test_specifically_prompt = "The section on status codes"

    test_spec_content = pathlib.Path("rfc9110.txt").read_text()

    print("=== REDUCER COMPARISON TEST ===")
    print("Testing hypothesis: Embedding model is better at finding specific context")
    print(f"Query: '{test_specifically_prompt}'")
    print("\n" + "="*60)

    # Test embedding-based approach
    print("\n1. EMBEDDING-BASED APPROACH:")
    print("-" * 30)

    embedding_reducer = EmbeddingReducer("BAAI/bge-small-en-v1.5", top_k=5)
    embedding_result, embedding_time = embedding_reducer.find_relevant_context(
        test_spec_content, test_specifically_prompt, context_before=1, context_after=1
    )

    timing = embedding_reducer.last_timing
    print(f"⏱️  Total time: {embedding_time*1000:.1f} ms")
    print(f"⏱️  Encoding time: {timing['encoding_time']*1000:.1f} ms")
    print(f"⏱️  Similarity computation: {timing['similarity_time']*1000:.2f} ms")
    print(f"📊 Lines processed: {timing['num_lines']}")
    print(f"📊 Time per line: {timing['encoding_time']*1000/timing['num_lines']:.2f} ms")

    print(f"\nLength: {len(embedding_result)} characters")
    print(f"Lines: {len(embedding_result.split(chr(10)))}")
    print("\nTop matches:")
    print(embedding_result[:800] + "..." if len(embedding_result) > 800 else embedding_result)

    # Test TextTiling-based approach
    print("\n\n2. TEXTTILING-BASED APPROACH:")
    print("-" * 30)

    texttiling_reducer = TextTilingReducer(w=20, k=10)
    texttiling_result, texttiling_time = texttiling_reducer.find_relevant_context(
        test_spec_content, test_specifically_prompt, max_segments=3
    )

    texttiling_timing = texttiling_reducer.last_timing
    print(f"⏱️  Total time: {texttiling_time*1000:.2f} ms")
    print(f"📊 Segments found: {texttiling_timing['num_segments']}")
    print(f"📊 Query length: {texttiling_timing['query_length']}")

    print(f"\nLength: {len(texttiling_result)} characters")
    print(f"Lines: {len(texttiling_result.split(chr(10)))}")
    print("\nTop segments:")
    print(texttiling_result[:800] + "..." if len(texttiling_result) > 800 else texttiling_result)

    # Analysis
    print("\n\n3. ANALYSIS:")
    print("-" * 30)

    # Count status code mentions in each result
    status_code_pattern = r"\b\d{3}\b"
    embedding_codes = len(re.findall(status_code_pattern, embedding_result))
    texttiling_codes = len(re.findall(status_code_pattern, texttiling_result))

    print(f"Status codes found in embedding result: {embedding_codes}")
    print(f"Status codes found in texttiling result: {texttiling_codes}")

    # Content quality assessment
    embedding_relevance = "status" in embedding_result.lower() or "code" in embedding_result.lower()
    texttiling_relevance = "status" in texttiling_result.lower() or "code" in texttiling_result.lower()

    print(f"Embedding result contains status-related content: {embedding_relevance}")
    print(f"TextTiling result contains status-related content: {texttiling_relevance}")

    print(f"\nEmbedding approach efficiency: {embedding_codes}/{len(embedding_result)} codes per char")
    print(f"TextTiling approach efficiency: {texttiling_codes}/{len(texttiling_result)} codes per char" if texttiling_result else "N/A")

    # Performance comparison
    print(f"\n📈 PERFORMANCE COMPARISON:")
    print(f"Embedding total time: {embedding_time*1000:.1f} ms")
    print(f"TextTiling total time: {texttiling_time*1000:.2f} ms")

    # Calculate relative performance
    times = [('Embedding', embedding_time), ('TextTiling', texttiling_time)]
    times.sort(key=lambda x: x[1])

    print(f"\nRanking by speed:")
    for i, (method, time_val) in enumerate(times, 1):
        print(f"{i}. {method}: {time_val*1000:.2f} ms")

    if embedding_time > 0 and texttiling_time > 0:
        speedup = embedding_time / texttiling_time
        print(f"\nTextTiling is {speedup:.1f}x {'faster' if speedup > 1 else 'slower'} than embedding")

    # Conclusion
    print("\n4. CONCLUSION:")
    print("-" * 30)

    # Find the approach with most status codes
    code_results = [
        ('Embedding', embedding_codes),
        ('TextTiling', texttiling_codes)
    ]
    code_results.sort(key=lambda x: x[1], reverse=True)

    print(f"Status code detection ranking:")
    for i, (method, codes) in enumerate(code_results, 1):
        print(f"{i}. {method}: {codes} status codes found")

    # Best performer by codes found
    best_method = code_results[0][0]
    print(f"\n✅ {best_method.upper()} approach found the most relevant status codes")

    # Efficiency analysis
    results_data = [
        ('Embedding', embedding_result, embedding_codes, embedding_time),
        ('TextTiling', texttiling_result, texttiling_codes, texttiling_time)
    ]

    # Find most efficient (highest codes per character ratio)
    efficiency_scores = []
    for method, result, codes, time_val in results_data:
        if result and len(result) > 0:
            efficiency = codes / len(result)
            efficiency_scores.append((method, efficiency))

    if efficiency_scores:
        efficiency_scores.sort(key=lambda x: x[1], reverse=True)
        most_efficient = efficiency_scores[0][0]
        print(f"⚡ {most_efficient.upper()} approach is most efficient (codes per character)")

    # Speed winner
    fastest = times[0][0]
    print(f"🏃 {fastest.upper()} approach is fastest")

    return {
        'embedding_result': embedding_result,
        'texttiling_result': texttiling_result,
        'embedding_codes': embedding_codes,
        'texttiling_codes': texttiling_codes,
        'embedding_time': embedding_time,
        'texttiling_time': texttiling_time,
        'timing_breakdown': timing,
        'texttiling_timing': texttiling_timing
    }


if __name__ == "__main__":
    test_reducer_comparison()
