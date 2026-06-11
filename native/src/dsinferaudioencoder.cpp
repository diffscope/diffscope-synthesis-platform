/**************************************************************************
 * DiffScope Synthesis Platform                                           *
 * Copyright (C) 2026 Team OpenVPI                                        *
 *                                                                        *
 * This program is free software: you can redistribute it and/or modify   *
 * it under the terms of the GNU General Public License as published by   *
 * the Free Software Foundation, either version 3 of the License, or      *
 * (at your option) any later version.                                    *
 *                                                                        *
 * This program is distributed in the hope that it will be useful,        *
 * but WITHOUT ANY WARRANTY; without even the implied warranty of         *
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the          *
 * GNU General Public License for more details.                           *
 *                                                                        *
 * You should have received a copy of the GNU General Public License      *
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. *
 **************************************************************************/

#include "dsinferdata.h"

#include <cstdint>
#include <cmath>
#include <limits>
#include <memory>
#include <stdexcept>
#include <vector>

#include <FLAC/stream_encoder.h>

namespace dssp {

	namespace {

		constexpr uint16_t WavFormatIEEEFloat = 3;
		constexpr uint16_t WavChannelCount = 1;
		constexpr uint16_t WavBitsPerSample = 32;
		constexpr uint16_t WavBlockAlign = WavChannelCount * WavBitsPerSample / 8;
		constexpr size_t WavHeaderSize = 44;

		void appendASCII(DiffSingerRawData &data, const char *value, size_t count) {
			for (size_t index = 0; index < count; ++index) {
				data.push_back(static_cast<uint8_t>(value[index]));
			}
		}

		void appendUint16LE(DiffSingerRawData &data, uint16_t value) {
			data.push_back(static_cast<uint8_t>(value));
			data.push_back(static_cast<uint8_t>(value >> 8));
		}

		void appendUint32LE(DiffSingerRawData &data, uint32_t value) {
			data.push_back(static_cast<uint8_t>(value));
			data.push_back(static_cast<uint8_t>(value >> 8));
			data.push_back(static_cast<uint8_t>(value >> 16));
			data.push_back(static_cast<uint8_t>(value >> 24));
		}

		bool isEncodable(const DiffSingerAudioData &audioData) {
			if (audioData.sampleRate <= 0) {
				return false;
			}
			if (audioData.audioData.size() % WavBlockAlign != 0) {
				return false;
			}

			const auto byteRate = static_cast<uint64_t>(audioData.sampleRate) * WavBlockAlign;
			return byteRate <= std::numeric_limits<uint32_t>::max();
		}

		DiffSingerRawData encodeWAV(const DiffSingerAudioData &audioData) {
			const auto dataSize = static_cast<uint32_t>(audioData.audioData.size());
			const auto riffSize = static_cast<uint32_t>(36 + audioData.audioData.size());
			const auto sampleRate = static_cast<uint32_t>(audioData.sampleRate);
			const auto byteRate = sampleRate * WavBlockAlign;

			DiffSingerRawData result;
			result.reserve(WavHeaderSize + audioData.audioData.size());

			appendASCII(result, "RIFF", 4);
			appendUint32LE(result, riffSize);
			appendASCII(result, "WAVE", 4);
			appendASCII(result, "fmt ", 4);
			appendUint32LE(result, 16);
			appendUint16LE(result, WavFormatIEEEFloat);
			appendUint16LE(result, WavChannelCount);
			appendUint32LE(result, sampleRate);
			appendUint32LE(result, byteRate);
			appendUint16LE(result, WavBlockAlign);
			appendUint16LE(result, WavBitsPerSample);
			appendASCII(result, "data", 4);
			appendUint32LE(result, dataSize);
			result.insert(result.end(), audioData.audioData.begin(), audioData.audioData.end());
			return result;
		}

		DiffSingerRawData encodeFLAC(const DiffSingerAudioData &audioData) {
			const auto buffer = reinterpret_cast<const float *>(audioData.audioData.data());
			const auto sampleCount = audioData.audioData.size() / WavBlockAlign;
			const auto sampleRate = audioData.sampleRate;
			std::vector<uint8_t> encodedData;

			struct EncoderDeleter {
				void operator()(FLAC__StreamEncoder *encoder) const {
					if (encoder) {
						FLAC__stream_encoder_delete(encoder);
					}
				}
			};

			std::unique_ptr<FLAC__StreamEncoder, EncoderDeleter> encoder(FLAC__stream_encoder_new());

			if (!encoder) {
				throw std::runtime_error("Failed to create FLAC encoder.");
			}

			constexpr unsigned Channels = 1;
			constexpr unsigned BitsPerSample = 24;
			constexpr FLAC__int32 Max24 =  8388607;
			constexpr FLAC__int32 Min24 = -8388608;

			FLAC__stream_encoder_set_channels(encoder.get(), Channels);
			FLAC__stream_encoder_set_bits_per_sample(encoder.get(), BitsPerSample);
			FLAC__stream_encoder_set_sample_rate(encoder.get(), sampleRate);

			FLAC__stream_encoder_set_compression_level(encoder.get(), 8);

			FLAC__stream_encoder_set_verify(encoder.get(), true);
			FLAC__stream_encoder_set_do_exhaustive_model_search(encoder.get(), true);

			FLAC__stream_encoder_set_total_samples_estimate(
				encoder.get(),
				static_cast<FLAC__uint64>(sampleCount)
			);

			auto writeCallback = [](
				const FLAC__StreamEncoder *,
				const FLAC__byte data[],
				size_t bytes,
				unsigned,
				unsigned,
				void *clientData
			) -> FLAC__StreamEncoderWriteStatus {
				auto *out = static_cast<std::vector<uint8_t> *>(clientData);

				const auto *begin = reinterpret_cast<const uint8_t *>(data);
				out->insert(out->end(), begin, begin + bytes);

				return FLAC__STREAM_ENCODER_WRITE_STATUS_OK;
			};

			auto initStatus = FLAC__stream_encoder_init_stream(
				encoder.get(),
				writeCallback,
				nullptr,   // seek callback
				nullptr,   // tell callback
				nullptr,   // metadata callback
				&encodedData
			);

			if (initStatus != FLAC__STREAM_ENCODER_INIT_STATUS_OK) {
				throw std::runtime_error("Failed to initialize FLAC encoder.");
			}

			const auto floatToPCM24 = [](float x) -> FLAC__int32 {
				if (!std::isfinite(x)) {
					return 0;
				}

				if (x >= 1.0f) {
					return Max24;
				}

				if (x <= -1.0f) {
					return Min24;
				}

				return static_cast<FLAC__int32>(std::lrint(x * static_cast<float>(Max24)));
			};

			constexpr size_t BlockSamples = 4096;
			std::vector<FLAC__int32> pcm;
			pcm.resize(BlockSamples);

			size_t processed = 0;

			while (processed < sampleCount) {
				const size_t blockSize = std::min(BlockSamples, sampleCount - processed);

				for (size_t i = 0; i < blockSize; ++i) {
					pcm[i] = floatToPCM24(buffer[processed + i]);
				}

				FLAC__bool ok = FLAC__stream_encoder_process_interleaved(
					encoder.get(),
					pcm.data(),
					static_cast<unsigned>(blockSize)
				);

				if (!ok) {
					FLAC__stream_encoder_finish(encoder.get());
					throw std::runtime_error("Failed to encode FLAC samples.");
				}

				processed += blockSize;
			}

			if (!FLAC__stream_encoder_finish(encoder.get())) {
				throw std::runtime_error("Failed to finish FLAC encoder.");
			}

			return encodedData;
		}

	} // namespace

} // namespace dssp

DSSP_DiffSingerRawData DSSP_EncodeWAV(DSSP_DiffSingerAudioData audioData) {
	const auto *data = dssp::getDiffSingerAudioData(audioData);
	if (data == nullptr || !dssp::isEncodable(*data)) {
		return nullptr;
	}

	return new dssp::DiffSingerRawData(dssp::encodeWAV(*data));
}

DSSP_DiffSingerRawData DSSP_EncodeFLAC(DSSP_DiffSingerAudioData audioData) {
	const auto *data = dssp::getDiffSingerAudioData(audioData);
	if (data == nullptr || !dssp::isEncodable(*data)) {
		return nullptr;
	}

	try {
		return new dssp::DiffSingerRawData(dssp::encodeFLAC(*data));
	} catch (const std::exception &) {
		return nullptr;
	}

}
