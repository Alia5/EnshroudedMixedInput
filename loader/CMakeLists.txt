cmake_minimum_required(VERSION 3.20)
project(emi_loader C RC)

add_library(dinput8 SHARED proxy.c)
set_target_properties(dinput8 PROPERTIES OUTPUT_NAME dinput8 PREFIX "")
target_compile_definitions(dinput8 PRIVATE WIN32_LEAN_AND_MEAN NOMINMAX)

# Static CRT: the loader must not depend on a VC++ redist being in the (Proton) prefix.
if(MSVC)
    set_property(TARGET dinput8 PROPERTY MSVC_RUNTIME_LIBRARY "MultiThreaded$<$<CONFIG:Debug>:Debug>")
    target_link_options(dinput8 PRIVATE "/DEF:${CMAKE_CURRENT_SOURCE_DIR}/proxy.def")
else()
    target_sources(dinput8 PRIVATE proxy.def)
    target_link_options(dinput8 PRIVATE -static)
endif()

set(EMI_REPO_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/..")
set_property(DIRECTORY APPEND PROPERTY CMAKE_CONFIGURE_DEPENDS "${EMI_REPO_ROOT}/versioninfo.json")
file(READ "${EMI_REPO_ROOT}/versioninfo.json" EMI_VERSIONINFO)

foreach(field Comments CompanyName FileVersion InternalName LegalCopyright LegalTrademarks PrivateBuild ProductName ProductVersion SpecialBuild)
    string(JSON EMI_${field} GET "${EMI_VERSIONINFO}" StringFileInfo ${field})
endforeach()

foreach(version FileVersion ProductVersion)
    set(numbers "")
    foreach(part Major Minor Patch Build)
        string(JSON number GET "${EMI_VERSIONINFO}" FixedFileInfo ${version} ${part})
        list(APPEND numbers ${number})
    endforeach()
    list(JOIN numbers "," EMI_FIXED_${version})
endforeach()

if(DEFINED EMI_VERSION)
    set(EMI_FileVersion "${EMI_VERSION}")
    set(EMI_ProductVersion "${EMI_VERSION}")
    set(EMI_FIXED_FileVersion "${EMI_FIXED_VERSION}")
    set(EMI_FIXED_ProductVersion "${EMI_FIXED_VERSION}")
endif()

string(JSON EMI_LANG_ID GET "${EMI_VERSIONINFO}" VarFileInfo Translation LangID)
string(JSON EMI_CHARSET_ID GET "${EMI_VERSIONINFO}" VarFileInfo Translation CharsetID)
string(JSON EMI_ICON_PATH ERROR_VARIABLE EMI_ICON_MISSING GET "${EMI_VERSIONINFO}" IconPath)
if(EMI_ICON_MISSING)
    set(EMI_ICON_LINE "")
else()
    cmake_path(ABSOLUTE_PATH EMI_ICON_PATH BASE_DIRECTORY "${EMI_REPO_ROOT}" NORMALIZE)
    set(EMI_ICON_LINE "1 ICON \"${EMI_ICON_PATH}\"")
endif()

configure_file(version.rc.in "${CMAKE_CURRENT_BINARY_DIR}/version.rc" @ONLY)
target_sources(dinput8 PRIVATE "${CMAKE_CURRENT_BINARY_DIR}/version.rc")

add_custom_command(TARGET dinput8 POST_BUILD
    COMMAND ${CMAKE_COMMAND} -E copy_if_different
        $<TARGET_FILE:dinput8>
        "${EMI_REPO_ROOT}/dist/dinput8.dll")
