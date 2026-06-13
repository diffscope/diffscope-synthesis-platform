cmake_minimum_required(VERSION 3.17)

project(diffscope-synthesis-platform VERSION 0.0.1 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

option(APPLICATION_INSTALL "Install DiffScope Synthesis Platform" ON)

if(MSVC)
	set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} /manifest:no")
	set(CMAKE_MODULE_LINKER_FLAGS "${CMAKE_MODULE_LINKER_FLAGS} /manifest:no")
	set(CMAKE_SHARED_LINKER_FLAGS "${CMAKE_SHARED_LINKER_FLAGS} /manifest:no")
	set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} /utf-8")
endif()

if(NOT DEFINED APPLICATION_NAME)
	set(APPLICATION_NAME "DiffScope Synthesis Platform")
endif()
if(NOT DEFINED APPLICATION_SEMVER)
	set(APPLICATION_SEMVER ${PROJECT_VERSION})
endif()

if(APPLICATION_INSTALL)
	include(GNUInstallDirs)
	include(CMakePackageConfigHelpers)
endif()

find_package(qmsetup CONFIG REQUIRED)

qm_import(Filesystem)
qm_import(Deploy)

if(NOT DEFINED QMSETUP_BUILD_DIR)
	set(QMSETUP_BUILD_DIR "${CMAKE_BINARY_DIR}/out")
endif()

qm_init_directories()

add_subdirectory(native)

find_program(GO_EXECUTABLE go REQUIRED)

set(APPLICATION_OUTPUT_DIR ${QMSETUP_BUILD_DIR}/bin)
set(APPLICATION_EXECUTABLE ${APPLICATION_OUTPUT_DIR}/dssp${CMAKE_EXECUTABLE_SUFFIX})
set(APPLICATION_GO_STAMP ${CMAKE_BINARY_DIR}/CMakeFiles/go_build.stamp)
set(APPLICATION_GO_EXTLDFLAGS "-L$<TARGET_FILE_DIR:dssp_native> -ldssp_native")

if(APPLE)
	string(APPEND APPLICATION_GO_EXTLDFLAGS " -Wl,-rpath,@executable_path/../lib")
elseif(UNIX)
	string(APPEND APPLICATION_GO_EXTLDFLAGS " -Wl,-rpath,$ORIGIN/../lib")
endif()

set(APPLICATION_GO_ENV
	"CGO_ENABLED=1"
	"CGO_CPPFLAGS=-I${CMAKE_SOURCE_DIR}/native/include"
	"CGO_CXXFLAGS=-I${CMAKE_SOURCE_DIR}/native/include"
	"CGO_LDFLAGS="
)

if(WIN32)
	list(APPEND APPLICATION_GO_ENV "CC=gcc" "CXX=g++")
endif()

add_custom_command(
	OUTPUT ${APPLICATION_GO_STAMP} "force_hook.fake"
	BYPRODUCTS ${APPLICATION_EXECUTABLE}
	DEPENDS dssp_native generate_swig_bindings
	COMMAND ${CMAKE_COMMAND} -E make_directory ${APPLICATION_OUTPUT_DIR}
	COMMAND
		${CMAKE_COMMAND} -E env
			${APPLICATION_GO_ENV}
			"${GO_EXECUTABLE}" build
				-ldflags "-X 'diffscope-synthesis-platform/internal/appinfo.ApplicationName=${APPLICATION_NAME}' -X 'diffscope-synthesis-platform/internal/appinfo.ApplicationSemver=${APPLICATION_SEMVER}' -extldflags '${APPLICATION_GO_EXTLDFLAGS}'"
				-o ${APPLICATION_EXECUTABLE}
				./cmd/dssp
	COMMAND ${CMAKE_COMMAND} -E touch ${APPLICATION_GO_STAMP}
	WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
	COMMENT "Building Go executable dssp"
	VERBATIM
)

set_source_files_properties("force_hook.fake" PROPERTIES SYMBOLIC TRUE)

add_custom_target(build_dssp ALL DEPENDS dssp_native generate_swig_bindings ${APPLICATION_GO_STAMP})

add_executable(dssp IMPORTED GLOBAL)
set_target_properties(dssp PROPERTIES IMPORTED_LOCATION ${APPLICATION_EXECUTABLE})

if(APPLICATION_INSTALL)
	install(PROGRAMS $<TARGET_FILE:dssp>
		DESTINATION "${CMAKE_INSTALL_BINDIR}"
	)

	if(WIN32)
		set(_lib_dir ${CMAKE_INSTALL_BINDIR})
		set(_extra_search_path ${QMSETUP_BUILD_DIR}/bin)
	else()
		set(_lib_dir "${CMAKE_INSTALL_LIBDIR}")
		set(_extra_search_path ${QMSETUP_BUILD_DIR}/lib)
	endif()

	qm_deploy_directory(${CMAKE_INSTALL_PREFIX}
		LIBRARY_DIR ${_lib_dir}
		EXTRA_SEARCHING_PATHS ${_extra_search_path}
		VERBOSE
	)

	if(MSVC AND APPLICATION_INSTALL_MSVC_RUNTIME)
		set(CMAKE_INSTALL_SYSTEM_RUNTIME_DESTINATION "${OUTPUT_DIR}")
		include(InstallRequiredSystemLibraries)
	endif()
endif()


