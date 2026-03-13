set(SOURCES
    main.cpp
    PasteApp.cpp
    HotkeyDialog.cpp
    UpdateChecker.cpp
)

set(HEADERS
    PasteApp.h
    HotkeyDialog.h
    UpdateChecker.h
    platform/InputSimulator.h
)

if(WIN32)
    list(APPEND SOURCES platform/InputSimulatorWin.cpp)
elseif(APPLE)
    list(APPEND SOURCES platform/InputSimulatorMac.cpp)
else()
    list(APPEND SOURCES platform/InputSimulatorLinux.cpp)
endif()

add_executable(paste_tool WIN32 ${SOURCES} ${HEADERS})

target_link_libraries(paste_tool PRIVATE
    paste_core
    Qt6::Widgets
    Qt6::Network
    qhotkey
)

target_include_directories(paste_tool PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_SOURCE_DIR}/core
)

if(WIN32)
    target_compile_definitions(paste_tool PRIVATE UNICODE _UNICODE)
endif()
