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

#ifndef DSSP_LUA_RUNNER_POOL_H
#define DSSP_LUA_RUNNER_POOL_H

#include <atomic>
#include <condition_variable>
#include <cstddef>
#include <functional>
#include <memory>
#include <mutex>
#include <optional>
#include <utility>
#include <vector>

namespace dssp {

	inline std::atomic_size_t &luaRunnerCountStorage() {
		static std::atomic_size_t count{1};
		return count;
	}

	inline std::size_t luaRunnerCount() {
		const auto count = luaRunnerCountStorage().load(std::memory_order_acquire);
		return count == 0 ? 1 : count;
	}

	inline void setLuaRunnerCount(std::size_t count) {
		luaRunnerCountStorage().store(count == 0 ? 1 : count, std::memory_order_release);
	}

	template <typename Runner>
	class LuaRunnerPool {
	public:
		class Lease {
		public:
			Lease() = default;

			Lease(LuaRunnerPool *pool, std::size_t index, std::size_t generation, Runner *runner)
				: m_pool(pool), m_index(index), m_generation(generation), m_runner(runner) {
			}

			~Lease() {
				release();
			}

			Lease(const Lease &) = delete;
			Lease &operator=(const Lease &) = delete;

			Lease(Lease &&other) noexcept
				: m_pool(other.m_pool), m_index(other.m_index), m_generation(other.m_generation), m_runner(other.m_runner) {
				other.m_pool = nullptr;
				other.m_runner = nullptr;
			}

			Lease &operator=(Lease &&other) noexcept {
				if (this != &other) {
					release();
					m_pool = other.m_pool;
					m_index = other.m_index;
					m_generation = other.m_generation;
					m_runner = other.m_runner;
					other.m_pool = nullptr;
					other.m_runner = nullptr;
				}
				return *this;
			}

			Runner *operator->() const {
				return m_runner;
			}

			Runner &operator*() const {
				return *m_runner;
			}

		private:
			void release() {
				if (m_pool) {
					m_pool->release(m_index, m_generation);
					m_pool = nullptr;
					m_runner = nullptr;
				}
			}

			LuaRunnerPool *m_pool{};
			std::size_t m_index{};
			std::size_t m_generation{};
			Runner *m_runner{};
		};

		template <typename Factory>
		LuaRunnerPool(std::size_t count, Factory factory)
			: m_factory(std::move(factory)) {
			if (count == 0) {
				count = 1;
			}
			m_runners.reserve(count);
			m_busy.resize(count, false);
			m_dirty.resize(count, false);
			for (std::size_t i = 0; i < count; ++i) {
				m_runners.push_back(m_factory());
			}
		}

		LuaRunnerPool(const LuaRunnerPool &) = delete;
		LuaRunnerPool &operator=(const LuaRunnerPool &) = delete;

		std::optional<Lease> acquire() {
			std::unique_lock lock(m_mutex);
			const auto generation = m_terminateGeneration;
			m_cv.wait(lock, [this, generation] {
				return m_terminateGeneration != generation || availableIndex().has_value();
			});

			if (m_terminateGeneration != generation) {
				return std::nullopt;
			}

			const auto index = *availableIndex();
			if (m_dirty.at(index)) {
				try {
					m_runners.at(index) = m_factory();
					m_dirty.at(index) = false;
				} catch (...) {
					m_cv.notify_all();
					throw;
				}
			}
			m_busy.at(index) = true;
			return Lease(this, index, m_terminateGeneration, m_runners.at(index).get());
		}

		void terminate() {
			{
				std::lock_guard lock(m_mutex);
				++m_terminateGeneration;
				for (std::size_t i = 0; i < m_runners.size(); ++i) {
					if (m_busy.at(i)) {
						m_runners.at(i)->interrupt();
					}
				}
			}
			m_cv.notify_all();
		}

	private:
		std::optional<std::size_t> availableIndex() const {
			for (std::size_t i = 0; i < m_busy.size(); ++i) {
				if (!m_busy.at(i)) {
					return i;
				}
			}
			return std::nullopt;
		}

		void release(std::size_t index, std::size_t generation) {
			{
				std::lock_guard lock(m_mutex);
				if (m_terminateGeneration != generation) {
					m_dirty.at(index) = true;
				}
				m_busy.at(index) = false;
			}
			m_cv.notify_one();
		}

		std::function<std::unique_ptr<Runner>()> m_factory;
		std::vector<std::unique_ptr<Runner>> m_runners;
		std::vector<bool> m_busy;
		std::vector<bool> m_dirty;
		std::mutex m_mutex;
		std::condition_variable m_cv;
		std::size_t m_terminateGeneration{};
	};

}

#endif // DSSP_LUA_RUNNER_POOL_H
